package functions

import (
	"regexp"
	"strings"
	"github.com/rihtim/core/log"
	"github.com/rihtim/core/utils"
	"github.com/rihtim/core/messages"
	"github.com/rihtim/core/requestscope"
	"github.com/rihtim/core/dataprovider"
)

type CoreFunctionController struct {
	functionHandlers map[string]FunctionHandler
}

func (cf *CoreFunctionController) Contains(path string) bool {

	for handlerPath := range cf.functionHandlers {
		validator, regExpErr := regexp.Compile(handlerPath)
		if !(strings.EqualFold(path, handlerPath) || (regExpErr == nil && validator.MatchString(path))) {
			continue
		}
		return true
	}
	return false
}

func (cf *CoreFunctionController) Add(path string, handler FunctionHandler) {

	path = utils.ConvertRichUrlToRegex(path, true)

	if cf.functionHandlers == nil {
		cf.functionHandlers = make(map[string]FunctionHandler)
	}
	cf.functionHandlers[path] = handler
	log.Debug("Function added for path: " + path)
}

func (cf *CoreFunctionController) Get(path string) (handler FunctionHandler, regex string) {

	for handlerPath, h := range cf.functionHandlers {
		validator, regExpErr := regexp.Compile(handlerPath)
		if !(strings.EqualFold(path, handlerPath) || (regExpErr == nil && validator.MatchString(path))) {
			continue
		}
		handler = h
		regex = handlerPath
		return
	}
	return
}

func (cf *CoreFunctionController) Execute(req messages.Message, rs requestscope.RequestScope, db dataprovider.Provider) (res messages.Message, editedRs requestscope.RequestScope, err *utils.Error) {

	functionHandler, regex := cf.Get(req.Res)

	// retrieve the url params and add into the request scope
	// ex: id from the url /users/{id}
	params, matches := utils.GetParamsFromRichUrl(regex, req.Res)

	if matches {
		for key, value := range params {
			rs.Set(key, value)
		}
	}

	res, rs, err = functionHandler(req, rs, db)
	return
}