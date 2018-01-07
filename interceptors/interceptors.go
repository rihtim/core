package interceptors

import (
	"github.com/rihtim/core/utils"
	"github.com/rihtim/core/messages"
	"github.com/rihtim/core/requestscope"
	"github.com/rihtim/core/dataprovider"
)

type Interceptor func(rs requestscope.RequestScope, extras interface{}, req, resp messages.Message, dp dataprovider.Provider) (editedReq, editedResp messages.Message, editedRs requestscope.RequestScope, err *utils.Error)

type InterceptorType int

const (
	BEFORE_EXEC InterceptorType = iota
	AFTER_EXEC
	ON_ERROR
	FINAL
)

const AnyPath = ".+"

var typeNames = [...]string{
	"BEFORE_EXEC",
	"AFTER_EXEC",
	"ON_ERROR",
	"FINAL",
}

type InterceptorController interface {
	Add(path, method string, iType InterceptorType, interceptor Interceptor, extras interface{})
	Get(res, method string, interceptorType InterceptorType) (interceptors []Interceptor, extras []interface{}, paths []string)
	Execute(res, method string, interceptorType InterceptorType, requestScope requestscope.RequestScope, request, response messages.Message, db dataprovider.Provider) (editedRequest, editedResponse messages.Message, editedRequestScope requestscope.RequestScope, err *utils.Error)
}
