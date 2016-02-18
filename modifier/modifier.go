package modifier

import (
	"strings"
	"reflect"
	"net/http"
	"gopkg.in/mgo.v2/bson"
	"github.com/rihtim/core/utils"
	"github.com/rihtim/core/database"
	"github.com/rihtim/core/constants"
)

var ExpandArray = func(data map[string]interface{}, config string) (result map[string]interface{}, err *utils.Error) {

	if !isValidExpandConfig(config) {
		err = &utils.Error{http.StatusBadRequest, "Expand config is not valid."}
		return
	}

	if data == nil {
		err = &utils.Error{http.StatusInternalServerError, "Data is nil."}
		return
	}

	dataArrayInterface, hasDataArray := data[constants.ListIdentifier]
	if !hasDataArray {
		err = &utils.Error{http.StatusInternalServerError, "Array not found at 'data' field."}
		return
	}
	dataArray := dataArrayInterface.([]map[string]interface{})

	resultArray := make([]map[string]interface{}, len(dataArray))
	for i, v := range dataArray {
		var expandedObject map[string]interface{}
		expandedObject, err = ExpandItem(map[string]interface{}(v), config)
		if err != nil {
			return
		}
		resultArray[i] = expandedObject
	}

	result = make(map[string]interface{})
	result[constants.ListIdentifier] = resultArray
	return
}

func ExpandItem(data map[string]interface{}, config string) (result map[string]interface{}, err *utils.Error) {

	if !isValidExpandConfig(config) {
		err = &utils.Error{http.StatusBadRequest, "Expand config is not valid."}
		return
	}

	fields := seperateFields(config)

	// expand direct children
	for _, field := range fields {

		directChildField, childsSubFields := getChildFieldAndSubFields(field)

		reference := data[directChildField]
		if t := reflect.TypeOf(reference); reference == nil || t == nil || t.Kind() != reflect.Map {
			continue
		}

		var expandedObject map[string]interface{}
		if isValidReference(reference) {
			expandedObject, err = fetchData(reference.(map[string]interface{}))
			if err != nil {
				return
			}
		} else {
			expandedObject = reference.(map[string]interface{})
		}

		// expanding children
		if len(childsSubFields) > 0 {

			var expandedChild map[string]interface{}
			expandedChild, err = ExpandItem(expandedObject, childsSubFields)
			if err != nil {
				return
			}
			_ = expandedChild
			//			expandedObject[trimmedField] = expandedChild
		}

		// TODO don't modify original object
		data[directChildField] = expandedObject
	}

	result = data
	return
}

var seperateFields = func(fields string) (result []string) {

	result = make([]string, 0)
	lastSplitIndex := 0
	childLevel := 0

	for i, r := range fields {
		c := string(r)
		if c == "(" {
			childLevel++
		} else if c == ")" {
			childLevel--;
		} else if c == "," && childLevel == 0 {
			childConfig := fields[lastSplitIndex:i]
			result = append(result, childConfig)
			lastSplitIndex = i + 1
		}
	}
	childConfig := fields[lastSplitIndex:]
	result = append(result, childConfig)

	return
}

var getChildFieldAndSubFields = func(config string) (field, subFields string) {
	if !strings.Contains(config, "(") {
		field = config
		subFields = ""
		return
	}
	field = config[0:strings.Index(config, "(")]
	subFields = config[strings.Index(config, "(") + 1:strings.LastIndex(config, ")")]
	return
}

var isValidExpandConfig = func(config string) bool {
	return strings.Count(config, "(") == strings.Count(config, ")")
}

var fetchData = func(data map[string]interface{}) (object map[string]interface{}, err *utils.Error) {
	fieldType := reflect.TypeOf(data[constants.IdIdentifier])

	var id string
	if fieldType.Kind() == reflect.String {
		id = data[constants.IdIdentifier].(string)
	} else {
		id = data[constants.IdIdentifier].(bson.ObjectId).Hex()
	}
	className := data["_class"].(string)

	object, err = database.Adapter.Get(className, id)
	if err != nil {
		return
	}
	return
}

var isValidReference = func(reference interface{}) (bool) {
	if reference == nil {
		return false
	}

	if t := reflect.TypeOf(reference); t == nil || t.Kind() != reflect.Map {
		return false
	}

	referenceAsMap := reference.(map[string]interface{})

	_type, hasType := referenceAsMap["_type"]
	_, hasId := referenceAsMap[constants.IdIdentifier]
	_, hasClass := referenceAsMap[constants.IdIdentifier]
	return len(referenceAsMap) == 3 && hasType && hasId && hasClass && _type == "reference"
}