package core

import (
	"strings"
	"net/http"
	"github.com/rihtim/core/utils"
	"github.com/rihtim/core/methods"
	"github.com/rihtim/core/messages"
	"github.com/rihtim/core/requestscope"
	"github.com/rihtim/core/dataprovider"
)

var AllowedMethodsOfResourceTypes = map[string]map[string]bool{
	"collection": {
		"get":  true,
		"post": true,
	},
	"model": {
		"put":    true,
		"delete": true,
		"get":    true,
	},
}

func Execute(request messages.Message, db dataprovider.Provider) (response messages.Message, updatedRequestscope requestscope.RequestScope, err *utils.Error) {

	// check if the method is allowed on the resource type
	var resourceType string
	resPartCount := len(strings.Split(request.Res, "/"))
	if resPartCount == 2 {
		resourceType = "collection"
	} else if resPartCount == 3 {
		resourceType = "model"
	} else {
		err = &utils.Error{
			Code:    http.StatusMethodNotAllowed,
			Message: "Invalid resource schema.",
		}
		return
	}

	allowedMethods := AllowedMethodsOfResourceTypes[resourceType]
	if isMethodAllowed := allowedMethods[strings.ToLower(request.Command)]; !isMethodAllowed {
		err = &utils.Error{
			Code:    http.StatusMethodNotAllowed,
			Message: "Method not allowed on the resource type.",
		}
		return
	}

	// execute request
	if strings.EqualFold(request.Command, methods.Post) {
		response, err = handlePost(request, db)
	} else if strings.EqualFold(request.Command, methods.Get) {
		response, err = handleGet(request, db)
	} else if strings.EqualFold(request.Command, methods.Put) {
		response, err = handlePut(request, db)
	} else if strings.EqualFold(request.Command, methods.Delete) {
		response, err = handleDelete(request, db)
	}

	return
}

var handlePost = func(request messages.Message, db dataprovider.Provider) (response messages.Message, err *utils.Error) {

	class := strings.Split(request.Res, "/")[1]

	// TODO: don't hard code the 'files' path
	if !strings.EqualFold(class, "files") {
		response.Body, err = db.Create(class, request.Body)
	} else {
		response.Body, err = db.CreateFile(request.ReqBodyRaw)
	}

	if err == nil {
		response.Status = http.StatusCreated
	}
	return
}

var handleGet = func(request messages.Message, db dataprovider.Provider) (response messages.Message, err *utils.Error) {

	class := strings.Split(request.Res, "/")[1]

	isFileClass := strings.EqualFold(class, "files")
	isModelActor := len(strings.Split(request.Res, "/")) == 3
	isCollectionActor := len(strings.Split(request.Res, "/")) == 2

	if isModelActor {
		id := request.Res[strings.LastIndex(request.Res, "/")+1:]
		if isFileClass {
			response.RawBody, err = db.GetFile(id) // get file by id
		} else {
			response.Body, err = db.Get(class, id) // get object by id
		}
	} else if isCollectionActor {
		response.Body, err = db.Query(class, request.Parameters) // query collection
	}

	if err != nil {
		return
	}
	return
}

var handlePut = func(request messages.Message, db dataprovider.Provider) (response messages.Message, err *utils.Error) {

	class := strings.Split(request.Res, "/")[1]
	id := request.Res[strings.LastIndex(request.Res, "/")+1:]
	response.Body, err = db.Update(class, id, request.Body)
	return
}

var handleDelete = func(request messages.Message, db dataprovider.Provider) (response messages.Message, err *utils.Error) {

	if len(strings.Split(request.Res, "/")) == 3 {
		// delete object
		class := strings.Split(request.Res, "/")[1]
		id := request.Res[strings.LastIndex(request.Res, "/")+1:]
		response.Body, err = db.Delete(class, id)
		if err == nil {
			response.Status = http.StatusNoContent
		}
	}
	return
}
