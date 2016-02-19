package coreinterceptors

import (
	"net/http"
	"github.com/rihtim/core/utils"
	"github.com/rihtim/core/messages"
)

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