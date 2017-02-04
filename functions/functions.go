package functions

import (
	"regexp"
	"strings"
	"github.com/rihtim/core/log"
	"github.com/rihtim/core/utils"
	"github.com/rihtim/core/messages"
	"github.com/rihtim/core/requestscope"
)

type FunctionHandler func(request messages.Message, requestScope requestscope.RequestScope) (response messages.Message, editedRequestScope requestscope.RequestScope, err *utils.Error)

var functionHandlers map[string]FunctionHandler

var AddFunctionHandler = func(path string, handler FunctionHandler) {

	path = utils.ConvertRichUrlToRegex(path, true)

	if functionHandlers == nil {
		functionHandlers = make(map[string]FunctionHandler)
	}
	functionHandlers[path] = handler
	log.Debug("Function added for path: " + path)
}

var ContainsHandler = func(path string) bool {

	for handlerPath, _ := range functionHandlers {
		validator, rexpErr := regexp.Compile(handlerPath)
		if !(strings.EqualFold(path, handlerPath) || (rexpErr == nil && validator.MatchString(path))) {
			continue
		}
		return true
	}
	return false
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

var ExecuteFunction = func(request messages.Message, requestScope requestscope.RequestScope) (response messages.Message, editedRequestScope requestscope.RequestScope, err *utils.Error) {
	functionHandler := GetFunctionHandler(request.Res)
	response, requestScope, err = functionHandler(request, requestScope)
	return
} 