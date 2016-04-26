package coreinterceptors

import (
	"strings"
	"net/http"
	"github.com/rihtim/core/utils"
	"github.com/rihtim/core/messages"
)

var AllowedPaths []string

var MethodNotAllowed = func(user map[string]interface{}, request, response messages.Message) (editedRequest, editedResponse messages.Message, err *utils.Error) {
	err = &utils.Error{http.StatusMethodNotAllowed, http.StatusText(http.StatusMethodNotAllowed)}
	return
}

var NotFound = func(user map[string]interface{}, request, response messages.Message) (editedRequest, editedResponse messages.Message, err *utils.Error) {
	err = &utils.Error{http.StatusNotFound, http.StatusText(http.StatusNotFound)}
	return
}

var PathValidator = func(user map[string]interface{}, request, response messages.Message) (editedRequest, editedResponse messages.Message, err *utils.Error) {

	if AllowedPaths == nil {
		return
	}

	for _, v := range AllowedPaths {
		if strings.Index(request.Res, v) != -1 {
			return
		}
	}

	err = &utils.Error{http.StatusBadRequest, "Path is not valid."}
	return
}