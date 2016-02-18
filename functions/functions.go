package functions

import (
	"github.com/rihtim/core/utils"
	"github.com/rihtim/core/messages"
	"github.com/Sirupsen/logrus"
)

type FunctionHandler func(user interface{}, message messages.Message) (response messages.Message, hookBody map[string]interface{}, err *utils.Error)

var functionHandlers map[string]FunctionHandler
var log = logrus.New()

var AddFunctionHandler = func(path string, handler FunctionHandler) {

	if functionHandlers == nil {
		functionHandlers = make(map[string]FunctionHandler)
	}
	functionHandlers[path] = handler
	log.Info("Function added for path: " + path)
}

var GetFunctionHandler = func(path string) (handler FunctionHandler) {
	return functionHandlers[path]
}