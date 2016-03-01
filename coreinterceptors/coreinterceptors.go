package coreinterceptors

import (
	"strings"
	"net/http"
	"github.com/rihtim/core/utils"
	"github.com/rihtim/core/messages"
	"github.com/rihtim/core/modifier"
	"github.com/rihtim/core/constants"
	"fmt"
)

var AllowedPaths []string

var Expander = func(user map[string]interface{}, message messages.Message) (response messages.Message, err *utils.Error) {

	//	log.Info("Expander interceptor called.")
	fmt.Println("expander")
	fmt.Println(message)
	response = message

	if message.Parameters["expand"] != nil {
		expandConfig := message.Parameters["expand"][0]
		fmt.Println(response.Body)
		if _, hasDataArray := response.Body[constants.ListIdentifier]; hasDataArray {
			response.Body, err = modifier.ExpandArray(response.Body, expandConfig)
		} else {
			response.Body, err = modifier.ExpandItem(response.Body, expandConfig)
		}
		fmt.Println(response.Body)
	}

	return
}

var MethodNotAllowed = func(user map[string]interface{}, message messages.Message) (response messages.Message, err *utils.Error) {
	err = &utils.Error{http.StatusMethodNotAllowed, http.StatusText(http.StatusMethodNotAllowed)}
	return
}

var NotFound = func(user map[string]interface{}, message messages.Message) (response messages.Message, err *utils.Error) {
	err = &utils.Error{http.StatusNotFound, http.StatusText(http.StatusNotFound)}
	return
}

var PathValidator = func(user map[string]interface{}, message messages.Message) (response messages.Message, err *utils.Error) {

	if AllowedPaths == nil {
		response = message
		return
	}

	for _, v := range AllowedPaths {
		if strings.Index(message.Res, v) != -1 {
			response = message
			return
		}
	}

	err = &utils.Error{http.StatusBadRequest, "Path is not valid."}

	return
}