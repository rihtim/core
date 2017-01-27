package coreinterceptors

import (
	"net/http"
	"github.com/rihtim/core/log"
	"github.com/rihtim/core/utils"
	"github.com/rihtim/core/messages"
	"github.com/rihtim/core/functions"
	"github.com/rihtim/core/requestscope"
)

var AllowedPaths []string

var MethodNotAllowed = func(requestScope requestscope.RequestScope, request, response messages.Message) (editedRequest, editedResponse messages.Message, editedRequestScope requestscope.RequestScope, err *utils.Error) {
	err = &utils.Error{http.StatusMethodNotAllowed, http.StatusText(http.StatusMethodNotAllowed)}
	return
}

var NotFound = func(requestScope requestscope.RequestScope, request, response messages.Message) (editedRequest, editedResponse messages.Message, editedRequestScope requestscope.RequestScope, err *utils.Error) {
	err = &utils.Error{http.StatusNotFound, http.StatusText(http.StatusNotFound)}
	return
}

var PathValidator = func(requestScope requestscope.RequestScope, request, response messages.Message) (editedRequest, editedResponse messages.Message, editedRequestScope requestscope.RequestScope, err *utils.Error) {

	log.Debug("Interceptor: PathValidator")

	if functions.ContainsHandler(request.Res) {
		return
	}

	if AllowedPaths == nil {
		return
	}

	for _, v := range AllowedPaths {
		_, matches := utils.GetParamsFromRichUrl(utils.ConvertRichUrlToRegex(v, true), request.Res)
		if matches {
			return
		}
	}

	err = &utils.Error{http.StatusBadRequest, "Path is not valid."}
	return
}