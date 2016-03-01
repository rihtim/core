package coreinterceptors

import (
	"strings"
	"reflect"
	"net/http"
	"gopkg.in/mgo.v2/bson"
	"github.com/rihtim/core/utils"
	"github.com/rihtim/core/messages"
	"github.com/rihtim/core/constants"
	"github.com/rihtim/core/actors"
)

var FilterConfig map[string]interface{}

var Expander = func(user map[string]interface{}, message messages.Message) (response messages.Message, err *utils.Error) {

	response = message
	if message.Parameters["expand"] != nil {

		expandConfig := message.Parameters["expand"][0]
		if resultsArray, hasResultsArray := response.Body[constants.ListIdentifier].([]map[string]interface{}); hasResultsArray {
			for i, item := range resultsArray {

				var expandedItem map[string]interface{}
				var expandErr *utils.Error
				expandedItem, expandErr = expandItem(item, message, expandConfig)
				if expandErr != nil {
					resultsArray[i] = map[string]interface{}{"code": expandErr.Code, "message": expandErr.Message}
				} else {
					resultsArray[i] = expandedItem
				}
			}
		} else {
			response.Body, err = expandItem(message.Body, message, expandConfig)
		}
	}

	/*if _, hasDataArray := response.Body["results"]; hasDataArray {
		response.Body, err = modifier.ExpandArray(response.Body, expandConfig)
	} else {
		response.Body, err = modifier.ExpandItem(response.Body, expandConfig)
	}*/

	return
}

var Filter = func(user map[string]interface{}, message messages.Message) (response messages.Message, err *utils.Error) {

	response = message
	if FilterConfig == nil {
		return
	}

	//	if message.Parameters["filter"] != nil {
	//		filterConfig := message.Parameters["filter"][0]

	class := strings.Split(message.Res, "/")[1]
	if resultsArray, hasResultsArray := response.Body[constants.ListIdentifier].([]map[string]interface{}); hasResultsArray {
		for i, item := range resultsArray {
			resultsArray[i] = filterItem(class, item)
		}
	} else {
		response.Body = filterItem(class, response.Body)
	}

	return
}

var filterItem = func(class string, item map[string]interface{}) (map[string]interface{}) {

	classFilterConfig, hasClassFilterConfig := FilterConfig[class]
	if !hasClassFilterConfig {
		return item
	}

	for key, _ := range classFilterConfig.(map[string]bool) {
		if _, containsKey := item[key]; containsKey {
			delete(item, key)
		}
	}

	return item
}

/*var expandArray = func(data map[string]interface{}, config string) (result map[string]interface{}, err *utils.Error) {

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
		err = &utils.Error{http.StatusInternalServerError, "Array not found at '" + constants.ListIdentifier + "' field."}
		return
	}
	dataArray := dataArrayInterface.([]map[string]interface{})

	resultArray := make([]map[string]interface{}, len(dataArray))
	for i, v := range dataArray {
		var expandedObject map[string]interface{}
		expandedObject, err = expandItem(map[string]interface{}(v), config)
		if err != nil {
			return
		}
		resultArray[i] = expandedObject
	}

	result = make(map[string]interface{})
	result[constants.ListIdentifier] = resultArray
	return
}*/
/*

func expandItem(data map[string]interface{}, config string) (result map[string]interface{}, err *utils.Error) {

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
			expandedChild, err = expandItem(expandedObject, childsSubFields)
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
*/

func expandItem(item map[string]interface{}, message messages.Message, config string) (result map[string]interface{}, err *utils.Error) {

	if !isValidExpandConfig(config) {
		err = &utils.Error{http.StatusBadRequest, "Expand config is not valid."}
		return
	}

	fields := seperateFields(config)

	// expand direct children
	for _, field := range fields {

		directChildField, childsSubFields := getChildFieldAndSubFields(field)

		reference := item[directChildField]
		if t := reflect.TypeOf(reference); reference == nil || t == nil || t.Kind() != reflect.Map {
			continue
		}

		var expandedObject map[string]interface{}
		if isValidReference(reference) {
			expandedObject, err = fetchData(reference.(map[string]interface{}), message)

			if err != nil {
				return
			}
		} else {
			expandedObject = reference.(map[string]interface{})
		}

		// expanding children
		if len(childsSubFields) > 0 {

			var expandedChild map[string]interface{}
			expandedChild, err = expandItem(expandedObject, message, childsSubFields)
			if err != nil {
				return
			}
			_ = expandedChild
			//			expandedObject[trimmedField] = expandedChild
		}

		// TODO don't modify original object
		item[directChildField] = expandedObject
	}

	result = item
	return
}

var isValidExpandConfig = func(config string) bool {
	return strings.Count(config, "(") == strings.Count(config, ")")
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

var fetchData = func(data map[string]interface{}, message messages.Message) (object map[string]interface{}, err *utils.Error) {

	fieldType := reflect.TypeOf(data[constants.IdIdentifier])

	var id string
	if fieldType.Kind() == reflect.String {
		id = data[constants.IdIdentifier].(string)
	} else {
		id = data[constants.IdIdentifier].(bson.ObjectId).Hex()
	}
	className := data["_class"].(string)

	res := "/" + className + "/" + id
	actor := actors.CreateActorForRes(res)

	requestWrapper := messages.RequestWrapper{}
	requestWrapper.Message.Res = res
	requestWrapper.Message.Command = constants.CommandGet
	requestWrapper.Message.Headers = message.Headers

	var response messages.Message
	response, err = actors.HandleRequest(&actor, requestWrapper)
	if err != nil {
		object = map[string]interface{}{"code": err.Code, "message": err.Message}
		return
	}
	object = response.Body
	return
}
/*
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
}*/