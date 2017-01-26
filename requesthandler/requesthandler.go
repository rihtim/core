package requesthandler

import (
	"strings"
	"net/http"
	"github.com/rihtim/core/messages"
	"github.com/rihtim/core/utils"
	"github.com/rihtim/core/constants"
	"github.com/rihtim/core/interceptors"
	"github.com/rihtim/core/database"
	"github.com/rihtim/core/validator"
	"github.com/rihtim/core/requestscope"
)

var restrictedFieldsForCreate = map[string]bool{
	constants.IdIdentifier: false,
	"createdAt": false,
	"updatedAt": false,
}

var restrictedFieldsForUpdate = map[string]bool{
	constants.IdIdentifier: false,
	constants.AclIdentifier: false,
	constants.RolesIdentifier: false,
	"createdAt": false,
	"updatedAt": false,
}

var HandleRequest = func(request messages.Message, requestScope requestscope.RequestScope) (response messages.Message, updatedRequestscope requestscope.RequestScope, err *utils.Error) {

	var editedRequest, editedResponse messages.Message
	var editedRequestScope requestscope.RequestScope

	// call interceptors before execution. return if response or error is not nil.
	editedRequest, editedResponse, editedRequestScope, err = interceptors.ExecuteInterceptors(request.Res, request.Command, interceptors.BEFORE_EXEC, requestScope, request, response)
	if err != nil || !editedResponse.IsEmpty() {
		response = editedResponse
		return
	}

	// update request if interceptor returned an editedRequest
	if !editedRequest.IsEmpty() {
		request = editedRequest
	}

	// update request scope if interceptor returned an editedRequestScope
	if !editedRequestScope.IsEmpty() {
		requestScope = editedRequestScope
	}

	// execute request
	if strings.EqualFold(request.Command, constants.CommandPost) {
		response, err = handlePost(request)
	} else if strings.EqualFold(request.Command, constants.CommandGet) {
		response, err = handleGet(request)
	} else if strings.EqualFold(request.Command, constants.CommandPut) {
		response, err = handlePut(request)
	} else if strings.EqualFold(request.Command, constants.CommandDelete) {
		response, err = handleDelete(request)
	}
	if err != nil {
		return
	}

	// call interceptors after execution. return value of the editedRequest is ignored because the execution is done
	_, editedResponse, editedRequestScope, err = interceptors.ExecuteInterceptors(request.Res, request.Command, interceptors.AFTER_EXEC, requestScope, request, response)
	if err != nil {
		return
	}

	// update request scope if interceptor returned an editedRequestScope
	if !editedRequestScope.IsEmpty() {
		requestScope = editedRequestScope
	}

	// replace the response with the given response but do not return.
	if !editedResponse.IsEmpty() {
		response = editedResponse
	}

	// call interceptors on final. all the return values are ignored because the final interceptor
	// doesn't have any effect on the request or response. it serves as trigger after the request
	go interceptors.ExecuteInterceptors(request.Res, request.Command, interceptors.FINAL, requestScope, request, response)
	return
}

var handlePost = func(request messages.Message) (response messages.Message, err *utils.Error) {

	err = validator.ValidateInputFields(restrictedFieldsForCreate, request.Body)
	if err != nil {
		return
	}

	class := strings.Split(request.Res, "/")[1]

	if (!strings.EqualFold(class, constants.ClassFiles)) {
		response.Body, err = database.Adapter.Create(class, request.Body)
	} else {
		response.Body, err = database.Adapter.CreateFile(request.ReqBodyRaw)
	}

	if err == nil {
		response.Status = http.StatusCreated
	}
	return
}

var handleGet = func(request messages.Message) (response messages.Message, err *utils.Error) {

	class := strings.Split(request.Res, "/")[1]

	isFileClass := strings.EqualFold(class, constants.ClassFiles)
	isModelActor := len(strings.Split(request.Res, "/")) == 3
	isCollectionActor := len(strings.Split(request.Res, "/")) == 2

	if isModelActor {
		id := request.Res[strings.LastIndex(request.Res, "/") + 1:]
		if isFileClass {
			response.RawBody, err = database.Adapter.GetFile(id)    // get file by id
		} else {
			response.Body, err = database.Adapter.Get(class, id)    // get object by id
		}
	} else if isCollectionActor {
		response.Body, err = database.Adapter.Query(class, request.Parameters)    // query collection
	}

	if err != nil {
		return
	}
	return
}

var handlePut = func(request messages.Message) (response messages.Message, err *utils.Error) {

	err = validator.ValidateInputFields(restrictedFieldsForUpdate, request.Body)
	if err != nil {
		return
	}

	class := strings.Split(request.Res, "/")[1]
	id := request.Res[strings.LastIndex(request.Res, "/") + 1:]
	response.Body, err = database.Adapter.Update(class, id, request.Body)
	return
}

var handleDelete = func(request messages.Message) (response messages.Message, err *utils.Error) {

	if len(strings.Split(request.Res, "/")) == 3 {
		// delete object
		class := strings.Split(request.Res, "/")[1]
		id := request.Res[strings.LastIndex(request.Res, "/") + 1:]
		response.Body, err = database.Adapter.Delete(class, id)
		if err == nil {
			response.Status = http.StatusNoContent
		}
	}
	return
}