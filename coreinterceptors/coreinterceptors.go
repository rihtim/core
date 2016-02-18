package coreinterceptors

import (
	"net/http"
	"github.com/rihtim/core/utils"
	"github.com/rihtim/core/messages"
	log "github.com/Sirupsen/logrus"
)

var Expander = func(user map[string]interface{}, message messages.Message) (response messages.Message, err *utils.Error) {

	//	log.Info("Expander interceptor called.")
	response = message
	return
}

var MethodNotAllowed = func(user map[string]interface{}, message messages.Message) (response messages.Message, err *utils.Error) {
	log.Error("Not allowed method is called.")
	err = &utils.Error{http.StatusMethodNotAllowed, http.StatusText(http.StatusMethodNotAllowed)}
	return
}