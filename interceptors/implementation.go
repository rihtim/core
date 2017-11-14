package interceptors

import (
	"regexp"
	"strings"
	"github.com/rihtim/core/log"
	"github.com/rihtim/core/utils"
	"github.com/rihtim/core/messages"
	"github.com/rihtim/core/requestscope"
	"github.com/rihtim/core/dataprovider"
	"runtime"
	"reflect"
)

type interceptorIndex struct {
	res             string
	method          string
	extras          interface{}
	interceptorType InterceptorType
	interceptor     Interceptor
}

type CoreInterceptorController struct {
	interceptorsMap []interceptorIndex
}

func (ci *CoreInterceptorController) Add(res, method string, interceptorType InterceptorType, interceptor Interceptor, extras interface{}) {

	res = utils.ConvertRichUrlToRegex(res, true)

	if ci.interceptorsMap == nil {
		ci.interceptorsMap = make([]interceptorIndex, 0)
	}

	index := interceptorIndex{res, method, extras, interceptorType, interceptor}
	ci.interceptorsMap = append(ci.interceptorsMap, index)

	identifier := strings.Join([]string{typeNames[int(interceptorType)], method, res}, ", ")
	log.Debug("Interceptor added for preferences: " + identifier)
}

func (ci *CoreInterceptorController) Get(res, method string, interceptorType InterceptorType) (interceptors []Interceptor, extras []interface{}, paths []string) {

	interceptors = make([]Interceptor, 0)
	extras = make([]interface{}, 0)
	paths = make([]string, 0)

	for _, index := range ci.interceptorsMap {

		// skip if interceptor type doesn't match
		if interceptorType != index.interceptorType {
			continue
		}
		// skip if resource doesn't match -or- resource is not '*' -or- resource doesn't match as regex
		validator, regExpErr := regexp.Compile(index.res)
		if !(strings.EqualFold(res, index.res) || strings.EqualFold("*", index.res) || (regExpErr == nil && validator.MatchString(res))) {
			continue
		}
		if !(strings.EqualFold(method, index.method) || strings.EqualFold("*", index.method)) {
			continue
		}
		interceptors = append(interceptors, index.interceptor)
		extras = append(extras, index.extras)
		paths = append(paths, index.res)
	}

	return
}

func (ci *CoreInterceptorController) Execute(res, method string, interceptorType InterceptorType, requestScope requestscope.RequestScope, request, response messages.Message, db dataprovider.Provider) (editedRequest, editedResponse messages.Message, editedRequestScope requestscope.RequestScope, err *utils.Error) {

	log.Debug("ExecuteInterceptors: " + method + " " + typeNames[int(interceptorType)] + " " + res)
	interceptors, extras, paths := ci.Get(res, method, interceptorType)

	var inputRequest, outputRequest, inputResponse, outputResponse messages.Message
	var inputRequestScope, outputRequestScope requestscope.RequestScope

	inputRequest = request
	inputResponse = response
	inputRequestScope = requestScope
	for i, interceptor := range interceptors {

		path := paths[i]
		extra := extras[i]

		interceptorName := runtime.FuncForPC(reflect.ValueOf(interceptor).Pointer()).Name()
		log.Debug("Executing Interceptor: " + interceptorName)

		// retrieve the url params and add into the request scope
		// ex: id from the url /users/{id}
		// regex := utils.ConvertRichUrlToRegex(path, true)
		params, matches := utils.GetParamsFromRichUrl(path, res)
		if matches {
			for key, value := range params {
				requestScope.Set(key, value)
			}
		}

		outputRequest, outputResponse, outputRequestScope, err = interceptor(inputRequestScope, extra, inputRequest, inputResponse, db)
		if err != nil {
			return
		}

		// output of the previous interceptor becomes the input of the next interceptor
		if !outputRequest.IsEmpty() {
			inputRequest = outputRequest
		}
		if !outputResponse.IsEmpty() {
			inputResponse = outputResponse

			// BEFORE_EXEC interceptors' editedResponse cuts the request. so skip the rest and return the response
			if interceptorType == BEFORE_EXEC {
				editedResponse = inputResponse
				return
			}
		}
		if !outputRequestScope.IsEmpty() {
			inputRequestScope = outputRequestScope
		}
	}
	editedRequest = inputRequest
	editedResponse = inputResponse
	editedRequestScope = inputRequestScope
	return
}
