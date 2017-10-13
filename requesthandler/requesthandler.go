package requesthandler

import (
	"strings"
	"net/http"
	"github.com/rihtim/core/messages"
	"github.com/rihtim/core/utils"
	"github.com/rihtim/core/constants"
	"github.com/rihtim/core/database"
	"bitbucket.org/mentornity/api/validator"
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

var AllowedMethodsOfResourceTypes = map[string]map[string]bool{
	"collection": {
		"get": true,
		"post": true,
	},
	"model": {
		"put": true,
		"delete": true,
		"get": true,
	},
};

var HandleRequest = func(request messages.Message, requestScope requestscope.RequestScope) (response messages.Message, updatedRequestscope requestscope.RequestScope, err *utils.Error) {

	// check if the method is allowed on the resource type
	var resourceType string
	resPartCount := len(strings.Split(request.Res, "/"))
	if resPartCount == 2 {
		resourceType = "collection"
	} else if resPartCount == 3 {
		resourceType = "model"
	} else {
		err = &utils.Error{http.StatusMethodNotAllowed, "Invalid resource schema."}
		return
	}

	allowedMethods := AllowedMethodsOfResourceTypes[resourceType]
	if isMethodAllowed := allowedMethods[strings.ToLower(request.Command)]; !isMethodAllowed {
		err = &utils.Error{http.StatusMethodNotAllowed, "Method not allowed on the resource type."}
		return
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