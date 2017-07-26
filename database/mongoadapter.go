package database

import (
	"io"
	"time"
	"reflect"
	"net/http"
	"encoding/json"
	"encoding/base64"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
	"github.com/rihtim/core/utils"
	"github.com/rihtim/core/constants"
)

type MongoAdapter struct {
	Addresses []string
	Database string
	AuthDatabase string
	Username string
	Password string

	session  *mgo.Session
	dialInfo mgo.DialInfo
}

func (ma *MongoAdapter) Init() (err *utils.Error) {

	if ma.Addresses == nil {
		err = &utils.Error{http.StatusInternalServerError, "Database 'addresses' must be specified."};
		return
	}

	ma.dialInfo = mgo.DialInfo{
		Addrs:    ma.Addresses,
		Database: ma.AuthDatabase,
		Username: ma.Username,
		Password: ma.Password,
	}

	return
}

func (ma *MongoAdapter) Connect() (err *utils.Error) {

	var dialErr error
	ma.session, dialErr = mgo.DialWithInfo(&ma.dialInfo)
	if dialErr != nil {
		err = &utils.Error{http.StatusInternalServerError, "Database connection failed. Reason: " + dialErr.Error()};
		return
	}

	return
}

func (ma MongoAdapter) Create(collection string, data map[string]interface{}) (response map[string]interface{}, err *utils.Error) {

	sessionCopy := ma.session.Copy()
	defer sessionCopy.Close()
	connection := sessionCopy.DB(ma.Database).C(collection)

	createdAt := int32(time.Now().Unix())
	if _, hasId := data[constants.IdIdentifier]; !hasId {
		id := bson.NewObjectId()
		data[constants.IdIdentifier] = id.Hex()
	}
	data["createdAt"] = createdAt
	data["updatedAt"] = createdAt

	insertError := connection.Insert(data)
	if insertError != nil {
		message := "Inserting item to database failed: " + insertError.Error()
		err = &utils.Error{http.StatusInternalServerError, message};
		return
	}

	response = map[string]interface{}{
		constants.IdIdentifier: data[constants.IdIdentifier],
		"createdAt": createdAt,
	}
	return
}

func (ma MongoAdapter) Get(collection string, id string) (response map[string]interface{}, err *utils.Error) {

	sessionCopy := ma.session.Copy()
	defer sessionCopy.Close()
	connection := sessionCopy.DB(ma.Database).C(collection)

	response = make(map[string]interface{})

	getErr := connection.FindId(id).One(&response)
	if getErr != nil {
		err = &utils.Error{http.StatusNotFound, "Object from '" + collection + "' with id '" + id + "' not found."};
		response = nil
		return
	}
	return
}

func (ma MongoAdapter) Query(collection string, parameters map[string][]string) (response map[string]interface{}, err *utils.Error) {

	sessionCopy := ma.session.Copy()
	defer sessionCopy.Close()
	connection := sessionCopy.DB(ma.Database).C(collection)

	response = make(map[string]interface{})

	if parameters["aggregate"] != nil && parameters["where"] != nil {
		err = &utils.Error{http.StatusBadRequest, "Where and aggregate parameters cannot be used at the same request."}
		return
	}

	var results []map[string]interface{}
	var getErr error

	whereParam, hasWhereParam, whereParamErr := extractJsonParameter(parameters, "where")
	aggregateParam, hasAggregateParam, aggregateParamErr := extractJsonParameter(parameters, "aggregate")
	sortParam, hasSortParam, sortParamErr := extractStringParameter(parameters, "sort")
	limitParam, _, limitParamErr := extractIntParameter(parameters, "limit")
	skipParam, _, skipParamErr := extractIntParameter(parameters, "skip")

	if aggregateParamErr != nil {err = aggregateParamErr}
	if whereParamErr != nil {err = whereParamErr}
	if sortParamErr != nil {err = sortParamErr}
	if limitParamErr != nil {err = limitParamErr}
	if skipParamErr != nil {err = skipParamErr}
	if err != nil {return}

	if hasWhereParam && hasAggregateParam {
		err = &utils.Error{http.StatusInternalServerError, "Aggregation cannot be used with where parameter."};
		return
	}

	if hasAggregateParam {
		getErr = connection.Pipe(aggregateParam).All(&results)
	} else {
		query := connection.Find(whereParam).Skip(skipParam).Limit(limitParam)
		if hasSortParam {
			query = query.Sort(sortParam)
		}
		getErr = query.All(&results)
	}

	if getErr != nil {
		err = &utils.Error{http.StatusInternalServerError, "Getting items failed. Reason: " + getErr.Error()};
		return
	}

	if results != nil {
		response["results"] = results
	} else {
		response["results"] = make([]map[string]interface{}, 0)
	}
	return
}

func (ma MongoAdapter) Update(collection string, id string, data map[string]interface{}) (response map[string]interface{}, err *utils.Error) {

	sessionCopy := ma.session.Copy()
	defer sessionCopy.Close()
	connection := sessionCopy.DB(ma.Database).C(collection)

	if data == nil {
		err = &utils.Error{http.StatusBadRequest, "Request body cannot be empty for update requests."}
		return
	}

	data["updatedAt"] = int32(time.Now().Unix())

	objectToUpdate := make(map[string]interface{})
	findErr := connection.FindId(id).One(&objectToUpdate)
	if findErr != nil {
		err = &utils.Error{http.StatusNotFound, "Item not found."};
		return
	}

	// updating the fields that request body contains
	for k, v := range data {
		objectToUpdate[k] = v
	}

	updateErr := connection.UpdateId(id, objectToUpdate)
	if updateErr != nil {
		err = &utils.Error{http.StatusInternalServerError, "Update request to db failed."};
		return
	}

	response = map[string]interface{}{
		"updatedAt": data["updatedAt"],
	}
	return
}

func (ma MongoAdapter) Delete(collection string, id string) (response map[string]interface{}, err *utils.Error) {

	sessionCopy := ma.session.Copy()
	defer sessionCopy.Close()
	connection := sessionCopy.DB(ma.Database).C(collection)

	removeErr := connection.RemoveId(id)
	if removeErr != nil {
		err = &utils.Error{http.StatusNotFound, "Item not found."};
	}
	return
}

func (ma MongoAdapter) CreateFile(data io.ReadCloser) (response map[string]interface{}, err *utils.Error) {

	sessionCopy := ma.session.Copy()
	defer sessionCopy.Close()

	objectId := bson.NewObjectId()
	now := time.Now()
	fileName := objectId.Hex()

	gridFile, mongoErr := sessionCopy.DB(ma.Database).GridFS("fs").Create(fileName)
	if mongoErr != nil {
		err = &utils.Error{http.StatusInternalServerError, "Creating file failed. Reason: " + mongoErr.Error()}
		return
	}
	gridFile.SetId(fileName)
	gridFile.SetName(fileName)
	gridFile.SetUploadDate(now)

	dec := base64.NewDecoder(base64.StdEncoding, data)
	_, copyErr := io.Copy(gridFile, dec)
	if copyErr != nil {
		err = &utils.Error{http.StatusInternalServerError, "Writing file failed. Reason: " + copyErr.Error()}
		return
	}

	closeErr := gridFile.Close()
	if closeErr != nil {
		err = &utils.Error{http.StatusInternalServerError, "Closing file failed. Reason: " + closeErr.Error()}
		return
	}

	response = make(map[string]interface{})
	response[constants.IdIdentifier] = fileName
	response["createdAt"] = int32(now.Unix())
	return

	return
}

func (ma MongoAdapter) GetFile(id string) (response []byte, err *utils.Error) {

	sessionCopy := ma.session.Copy()
	defer sessionCopy.Close()


	file, mongoErr := sessionCopy.DB(ma.Database).GridFS("fs").OpenId(id)
	if mongoErr != nil {
		if mongoErr == mgo.ErrNotFound {
			err = &utils.Error{http.StatusNotFound, "File not found. Reason: " + mongoErr.Error()};
		} else {
			err = &utils.Error{http.StatusInternalServerError, "Getting file failed. Reason: " + mongoErr.Error()};
		}
		return
	}

	response = make([]byte, file.Size())
	_, printErr := file.Read(response)
	if printErr != nil {
		err = &utils.Error{http.StatusInternalServerError, "Printing file failed. Reason: " + printErr.Error()};
	}
	file.Close()
	return
}

var extractJsonParameter = func(parameters map[string][]string, key string) (value interface{}, hasParam bool, err *utils.Error) {

	var paramArray []string
	paramArray, hasParam = parameters[key]

	if hasParam {
		parseErr := json.Unmarshal([]byte(paramArray[0]), &value)
		if parseErr != nil {
			err = &utils.Error{http.StatusBadRequest, "Parsing " + key + " parameter failed. Reason: " + parseErr.Error()}
		}
	}
	return
}

var extractStringParameter = func(parameters map[string][]string, key string) (value string, hasParam bool, err *utils.Error) {

	var paramArray []string
	paramArray, hasParam = parameters[key]

	if hasParam {
		var paramValue interface{}
		parseErr := json.Unmarshal([]byte(paramArray[0]), &paramValue)
		if parseErr != nil {
			err = &utils.Error{http.StatusBadRequest, "Parsing " + key + " parameter failed. Reason: " + parseErr.Error()}
		}

		fieldType := reflect.TypeOf(paramValue)
		if fieldType == nil || fieldType.Kind() != reflect.String {
			value = ""
			err = &utils.Error{http.StatusBadRequest, "The key '" + key + "' must be a valid string."}
			return
		}
		value = paramValue.(string)
	}
	return
}

var extractIntParameter = func(parameters map[string][]string, key string) (value int, hasParam bool, err *utils.Error) {

	var paramArray []string
	paramArray, hasParam = parameters[key]

	if hasParam {
		var paramValue interface{}
		parseErr := json.Unmarshal([]byte(paramArray[0]), &paramValue)
		if parseErr != nil {
			err = &utils.Error{http.StatusBadRequest, "Parsing " + key + " parameter failed. Reason: " + parseErr.Error()}
		}

		fieldType := reflect.TypeOf(paramValue)
		if fieldType == nil || fieldType.Kind() != reflect.Float64 {
			value = 0
			err = &utils.Error{http.StatusBadRequest, "The key '" + key + "' must be an integer."}
			return
		}
		value = int(paramValue.(float64))
	}
	return
}
