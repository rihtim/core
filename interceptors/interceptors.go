package interceptors

import (
	"regexp"
	"strings"
	"github.com/rihtim/core/utils"
	"github.com/rihtim/core/messages"
"github.com/rihtim/core/log"
)

type Interceptor func(user map[string]interface{}, message messages.Message) (response messages.Message, err *utils.Error)

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
}

var interceptorsMap map[interceptorIndex]Interceptor

var AddInterceptor = func(res, method string, interceptorType InterceptorType, interceptor Interceptor) {

	if interceptorsMap == nil {
		interceptorsMap = make(map[interceptorIndex]Interceptor)
	}

	index := interceptorIndex{res, method, interceptorType}
	interceptorsMap[index] = interceptor

	identifier := strings.Join([]string{typeNames[int(interceptorType)], method, res}, ", ")
	log.Info("Interceptor added for preferences: " + identifier)
}

var GetInterceptor = func(res, method string, interceptorType InterceptorType) (interceptors []Interceptor) {

	interceptors = make([]Interceptor, 0)
	for index, interceptor := range interceptorsMap {
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
		interceptors = append(interceptors, interceptor)
	}

	return interceptors
}

var ExecuteInterceptors = func(res, method string, interceptorType InterceptorType, user map[string]interface{}, message messages.Message) (response messages.Message, err *utils.Error) {

	interceptors := GetInterceptor(res, method, interceptorType)

	var input, output messages.Message
	input = message
	for _, interceptor := range interceptors {
		output, err = interceptor(user, input)
		if err != nil {
			return
		}
		input = output    // output of the previous interceptor becomes the input of the next interceptor
	}
	response = input
	return
}
