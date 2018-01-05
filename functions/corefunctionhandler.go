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

type FunctionWrapper struct {
	path     string
	method   string
	extras   interface{}
	function FunctionHandler
}

type CoreFunctionController struct {
	functionHandlers []FunctionWrapper
}

func (cfc *CoreFunctionController) Contains(path, method string) bool {
	return cfc.FindIndex(path, method) != -1
}

func (cfc *CoreFunctionController) FindIndex(path, method string) int {

	for i, functionIndex := range cfc.functionHandlers {

		if method != functionIndex.method {
			// skip if method doesn't match
			continue
		}

		validator, regExpErr := regexp.Compile(functionIndex.path)
		if !(path == functionIndex.path || (regExpErr == nil && validator.MatchString(path))) {
			// skip if path and regExp comparison doesn't match
			continue
		}
		return i
	}
	return -1
}

func (cfc *CoreFunctionController) Add(path, method string, handler FunctionHandler, extras interface{}) {

	path = utils.ConvertRichUrlToRegex(path, true)

	if cfc.functionHandlers == nil {
		cfc.functionHandlers = make([]FunctionWrapper, 0)
	}

	index := FunctionWrapper{path, method, extras, handler}
	cfc.functionHandlers = append(cfc.functionHandlers, index)

	log.Debug("Function added for preferences: " + strings.Join([]string{method, path}, ", "))
}

func (cfc *CoreFunctionController) Execute(req messages.Message, rs requestscope.RequestScope, db dataprovider.Provider) (resp messages.Message, editedRs requestscope.RequestScope, err *utils.Error) {

	// copy request scope to prevent modification on input value
	editedRs = rs.Copy()

	// get function handler
	functionWrapper := cfc.functionHandlers[cfc.FindIndex(req.Res, req.Command)]

	// retrieve the url params and add into the request scope
	// ex: id from the url /users/{id}
	if params, matches := utils.GetParamsFromRichUrl(functionWrapper.path, req.Res); matches {
		for key, value := range params {
			editedRs.Set(key, value)
		}
	}

	// execute function handler
	resp, rsFromFunction, err := functionWrapper.function(req, editedRs, functionWrapper.extras, db)

	// assign request scope returned from function to editedRs
	if !rsFromFunction.IsEmpty() {
		editedRs = rsFromFunction
	}
	return
}
