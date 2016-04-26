package interceptors

import (
	"regexp"
	"strings"
	"github.com/rihtim/core/utils"
	"github.com/rihtim/core/messages"
	"github.com/rihtim/core/log"
)

//type Interceptor func(user map[string]interface{}, message messages.Message) (response messages.Message, err *utils.Error)
type Interceptor func(user map[string]interface{}, request, response messages.Message) (editedRequest, editedResponse messages.Message, err *utils.Error)

type InterceptorType int

const (
	BEFORE_AUTH InterceptorType = iota
	BEFORE_EXEC
	AFTER_EXEC
	FINAL
)

var typeNames = [...]string{
	"BEFORE_AUTH",
	"BEFORE_EXEC",
	"AFTER_EXEC",
	"FINAL",
}

type interceptorIndex struct {
	res             string
	method          string
	interceptorType InterceptorType
	interceptor     Interceptor
}

var interceptorsMap []interceptorIndex

var AddInterceptor = func(res, method string, interceptorType InterceptorType, interceptor Interceptor) {

	if interceptorsMap == nil {
		interceptorsMap = make([]interceptorIndex, 0)
	}

	index := interceptorIndex{res, method, interceptorType, interceptor}
	interceptorsMap = append(interceptorsMap, index)

	identifier := strings.Join([]string{typeNames[int(interceptorType)], method, res}, ", ")
	log.Info("Interceptor added for preferences: " + identifier)
}

var GetInterceptor = func(res, method string, interceptorType InterceptorType) (interceptors []Interceptor) {

	interceptors = make([]Interceptor, 0)
	for _, index := range interceptorsMap {
		// skip if interceptor type doesn't match
		if interceptorType != index.interceptorType {
			continue
		}
		// skip if resource doesn't match -or- resource is not '*' -or- resource doesn't match as regex
		validator, rexpErr := regexp.Compile(index.res)
		if !(strings.EqualFold(res, index.res) || strings.EqualFold("*", index.res) || (rexpErr == nil && validator.MatchString(res))) {
			continue
		}
		if !(strings.EqualFold(method, index.method) || strings.EqualFold("*", index.method)) {
			continue
		}
		interceptors = append(interceptors, index.interceptor)
	}

	return interceptors
}

var ExecuteInterceptors = func(res, method string, interceptorType InterceptorType, user map[string]interface{}, request, response messages.Message) (editedRequest, editedResponse messages.Message, err *utils.Error) {

	interceptors := GetInterceptor(res, method, interceptorType)

	var inputRequest, outputRequest, inputResponse, outputResponse messages.Message
	inputRequest = request
	inputResponse = response
	for _, interceptor := range interceptors {
		outputRequest, outputResponse, err = interceptor(user, inputRequest, inputResponse)
		if err != nil {
			return
		}

		// output of the previous interceptor becomes the input of the next interceptor
		if !outputRequest.IsEmpty() {
			inputRequest = outputRequest
		}
		if !outputResponse.IsEmpty() {
			inputResponse = outputResponse
		}
	}
	editedRequest = inputRequest
	editedResponse = inputResponse
	return
}
