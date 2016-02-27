package coreinterceptors

import (
	"net/http"
	"github.com/rihtim/core/utils"
	"github.com/rihtim/core/messages"
	"strings"
)

var AllowedPaths []string

var Expander = func(user map[string]interface{}, message messages.Message) (response messages.Message, err *utils.Error) {

	//	log.Info("Expander interceptor called.")
	response = message
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