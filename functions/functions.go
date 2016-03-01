package functions

import (
	"github.com/rihtim/core/utils"
	"github.com/rihtim/core/messages"
	"github.com/rihtim/core/log"
	"regexp"
"strings"
)

type FunctionHandler func(user interface{}, message messages.Message) (response messages.Message, finalInterceptorBody map[string]interface{}, err *utils.Error)

var functionHandlers map[string]FunctionHandler

var AddFunctionHandler = func(path string, handler FunctionHandler) {

	if functionHandlers == nil {
		functionHandlers = make(map[string]FunctionHandler)
	}
	functionHandlers[path] = handler
	log.Info("Function added for path: " + path)
}

var GetFunctionHandler = func(path string) (handler FunctionHandler) {

	for handlerPath, h := range functionHandlers {
		validator, rexpErr := regexp.Compile(handlerPath)
		if !(strings.EqualFold(path, handlerPath) || (rexpErr == nil && validator.MatchString(path))) {
			continue
		}
		handler = h
		return
	}
	return
}