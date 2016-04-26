package actors

import (
	"time"
	"strings"
	"net/http"
	"encoding/json"
	"github.com/rihtim/core/messages"
	"github.com/rihtim/core/utils"
	"github.com/rihtim/core/constants"
	"github.com/rihtim/core/functions"
	"github.com/rihtim/core/auth"
	"github.com/rihtim/core/interceptors"
	"github.com/rihtim/core/database"
	"github.com/rihtim/core/log"
	"github.com/Sirupsen/logrus"
	"github.com/rihtim/core/keys"
	"github.com/rihtim/core/validator"
)

type Actor struct {
	res         string
	actorType   string
	class       string
	model       map[string]interface{}
	children    map[string]Actor
	Inbox       chan messages.RequestWrapper
	parentInbox chan messages.RequestWrapper
	//	adapter     *adapters.MongoAdapter
}

var AllowedMethodsOfActorTypes = map[string]map[string]bool{
	"collection": {
		"get": true,
		"post": true,
	},
	"model": {
		"put": true,
		"delete": true,
		"get": true,
	},
	"functions": {
		"post": true,
	},
};

var restrictedFieldsForUpdate = map[string]bool{
	constants.IdIdentifier: false,
	constants.AclIdentifier: false,
	constants.RolesIdentifier: false,
	"createdAt": false,
	"updatedAt": false,
}

var restrictedFieldsForCreate = map[string]bool{
	constants.IdIdentifier: false,
	"createdAt": false,
	"updatedAt": false,
}

var CreateActor = func(parent *Actor, res string) (a Actor) {
	//	log.Debug("Creating actor for " + res)

	if parent == nil {
		a.actorType = constants.ActorTypeRoot
	} else if functionHandler := functions.GetFunctionHandler(res); functionHandler != nil {
		a.actorType = constants.ActorTypeFunctions
	} else if strings.EqualFold(parent.actorType, constants.ActorTypeRoot) {
		a.actorType = constants.ActorTypeCollection
		resParts := strings.Split(res, "/")
		a.class = resParts[1]
	} else if strings.EqualFold(parent.actorType, constants.ActorTypeCollection) {
		a.actorType = constants.ActorTypeModel
		a.class = parent.class
	}

	a.res = res
	a.children = make(map[string]Actor)
	a.Inbox = make(chan messages.RequestWrapper)
	if parent != nil {
		a.parentInbox = parent.Inbox
	}
	return
}

var CreateActorForRes = func(res string) (a Actor) {
	//	log.Debug("Creating actor for " + res)

	if functionHandler := functions.GetFunctionHandler(res); functionHandler != nil {
		a.actorType = constants.ActorTypeFunctions
	} else if resParts := strings.Split(res, "/"); len(resParts) == 2 {
		a.actorType = constants.ActorTypeCollection
		a.class = resParts[1]

	} else if len(resParts) == 3 {
		a.actorType = constants.ActorTypeModel
		a.class = resParts[1]
	}

	a.res = res
	a.Inbox = make(chan messages.RequestWrapper)
	return
}

func (a *Actor) Run() {
	defer func() {
		log.Debug(a.res + ":  Stopped running.")
	}()
	log.Debug(a.res + ":  Started running.")

	for {
		select {
		case requestWrapper := <-a.Inbox:
			log.Debug(a.res + ": Received a message.")

			if requestWrapper.Message.Res == a.res {
				// if the resource of the message is this actor's resource

				messageString, mErr := json.Marshal(requestWrapper.Message)
				if mErr == nil {
					log.Info(a.res + ": Handling " + string(messageString))
				}

				response, err := HandleRequest(a, requestWrapper)

				if err != nil {
					if response.Status == 0 {
						response.Status = err.Code
					}
					if response.Body == nil {
						response.Body = map[string]interface{}{"message":err.Message}
					}
					log.Error(err.Error())
				}

				a.checkAndSend(requestWrapper.Listener, response)

				responseString, rmErr := json.Marshal(response.Body)
				if rmErr == nil {
					log.Info(a.res + ": Responding " + string(responseString))
				}

				// TODO stop the actor if it belongs to an item and the item is deleted
				// TODO stop the actor if it belongs to an item and the item doesn't exist
				// TODO stop the actor if it belongs to an entity and 'get' returns an empty array (not sure though)

			} else {
				log.Info(a.res + ": Forwarding message to child actor.")

				// if the resource belongs to a children actor
				childRes := getChildRes(requestWrapper.Message.Res, a.res)

				actor, exists := a.children[childRes]
				if !exists {
					// if children doesn't exists, create a child actor for the res
					actor = CreateActor(a, childRes)
					go actor.Run()
					a.children[childRes] = actor
				}
				//   forward message to the children actor
				actor.Inbox <- requestWrapper
			}
		}
	}
}

var HandleRequest = func(a *Actor, requestWrapper messages.RequestWrapper) (response messages.Message, err *utils.Error) {

	start := time.Now()
	log.WithFields(logrus.Fields{
		"res": requestWrapper.Message.Res,
		"command": requestWrapper.Message.Command,
	}).Info("Received request.")

	// check for method is allowed on the resource type except functions
	if !strings.EqualFold(a.actorType, constants.ActorTypeFunctions) {
		allowedMethods := AllowedMethodsOfActorTypes[a.actorType]
		if isMethodAllowed := allowedMethods[strings.ToLower(requestWrapper.Message.Command)]; !isMethodAllowed {
			err = &utils.Error{http.StatusMethodNotAllowed, http.StatusText(http.StatusMethodNotAllowed)}
			return
		}
	}
	request := requestWrapper.Message
	var editedRequest, editedResponse messages.Message

	// call interceptors before authentication. return if response or error is not nil.
	editedRequest, editedResponse, err = interceptors.ExecuteInterceptors(request.Res, request.Command, interceptors.BEFORE_AUTH, nil, request, response)
	if err != nil || !editedResponse.IsEmpty() {
		response = editedResponse
		return
	}

	// update request if interceptor returned updatedRequest
	if !editedRequest.IsEmpty() {
		request = editedRequest
	}

	// check whether the headers give special permissions to perform the request
	var isGrantedByKey bool
	isGrantedByKey, err = keys.Adapter.CheckKeyPermissions(requestWrapper.Message.Headers)
	if err != nil {
		return
	}

	// check permissions of user if request is not granted by keys.
	// continue anyway if the actor type is ActorTypeFunctions because the security should be handled in function itself
	var user map[string]interface{}
	if !isGrantedByKey && !strings.EqualFold(a.actorType, constants.ActorTypeFunctions) {
		var isGranted bool
		isGranted, user, err = auth.IsGranted(a.class, requestWrapper)
		if !isGranted {
			if err == nil {
				err = &utils.Error{http.StatusUnauthorized, http.StatusText(http.StatusUnauthorized)}
			}
			return
		}
	}

	// call interceptors before execution. return if response or error is not nil.
	editedRequest, editedResponse, err = interceptors.ExecuteInterceptors(request.Res, request.Command, interceptors.BEFORE_EXEC, user, request, response)
	if err != nil || !editedResponse.IsEmpty() {
		response = editedResponse
		return
	}

	// update request if interceptor returned updatedRequest
	if !editedRequest.IsEmpty() {
		request = editedRequest
	}
	requestWrapper.Message = request

	// execute request
	if (strings.EqualFold(a.actorType, constants.ActorTypeFunctions)) {
		functionHandler := functions.GetFunctionHandler(request.Res)
		response, _, err = functionHandler(user, request)
	} else if strings.EqualFold(requestWrapper.Message.Command, constants.CommandPost) {
		response, _, err = handlePost(a, requestWrapper, user)
	} else if strings.EqualFold(requestWrapper.Message.Command, constants.CommandGet) {
		response, err = handleGet(a, requestWrapper)
	} else if strings.EqualFold(requestWrapper.Message.Command, constants.CommandPut) {
		response, _, err = handlePut(a, requestWrapper)
	} else if strings.EqualFold(requestWrapper.Message.Command, constants.CommandDelete) {
		response, err = handleDelete(a, requestWrapper)
	}
	if err != nil {
		return
	}

	// call interceptors after execution. return value of the editedRequest is ignored because the execution is done
	_, editedResponse, err = interceptors.ExecuteInterceptors(request.Res, request.Command, interceptors.AFTER_EXEC, user, request, response)
	if err != nil {
		return
	}

	// replace the response with the given response but do not return.
	if !editedResponse.IsEmpty() {
		response = editedResponse
	}

	// call interceptors on final. all the return values are ignored because the final interceptor
	// doesn't have any effect on the request or response. it serves as trigger after the request
	go interceptors.ExecuteInterceptors(request.Res, request.Command, interceptors.FINAL, user, request, response)

	elapsed := time.Since(start)
	log.WithFields(logrus.Fields{
		"res": requestWrapper.Message.Res,
		"command": requestWrapper.Message.Command,
		"duration": elapsed.Nanoseconds() / (int64(time.Millisecond) / int64(time.Nanosecond)),
	}).Info("Returning response.")
	return
}

var handlePost = func(a *Actor, requestWrapper messages.RequestWrapper, user interface{}) (response messages.Message, finalInterceptorBody map[string]interface{}, err *utils.Error) {

	err = validator.ValidateInputFields(restrictedFieldsForCreate, requestWrapper.Message.Body)
	if err != nil {
		return
	}

	if (!strings.EqualFold(a.class, constants.ClassFiles)) {
		response.Body, finalInterceptorBody, err = database.Adapter.Create(a.class, requestWrapper.Message.Body)
	} else {
		response.Body, finalInterceptorBody, err = database.Adapter.CreateFile(requestWrapper.Message.ReqBodyRaw)
	}

	if err == nil {
		response.Status = http.StatusCreated
	}
	return
}

var handleGet = func(a *Actor, requestWrapper messages.RequestWrapper) (response messages.Message, err *utils.Error) {

	isFileClass := strings.EqualFold(a.class, constants.ClassFiles)
	isModelActor := strings.EqualFold(a.actorType, constants.ActorTypeModel)
	isCollectionActor := strings.EqualFold(a.actorType, constants.ActorTypeCollection)

	if isModelActor {
		id := requestWrapper.Message.Res[strings.LastIndex(requestWrapper.Message.Res, "/") + 1:]
		if isFileClass {
			response.RawBody, err = database.Adapter.GetFile(id)    // get file by id
		} else {
			response.Body, err = database.Adapter.Get(a.class, id)    // get object by id
		}
	} else if isCollectionActor {
		response.Body, err = database.Adapter.Query(a.class, requestWrapper.Message.Parameters)    // query collection
	}

	if err != nil {
		return
	}

	// TODO remove the part below and create a 'during' interceptor for expanding fields
	/*if requestWrapper.Message.Parameters["expand"] != nil {
		expandConfig := requestWrapper.Message.Parameters["expand"][0]
		if _, hasDataArray := response.Body["results"]; hasDataArray {
			response.Body, err = modifier.ExpandArray(response.Body, expandConfig)
		} else {
			response.Body, err = modifier.ExpandItem(response.Body, expandConfig)
		}
	}*/
	return
}

var handlePut = func(a *Actor, requestWrapper messages.RequestWrapper) (response messages.Message, finalInterceptorBody map[string]interface{}, err *utils.Error) {

	err = validator.ValidateInputFields(restrictedFieldsForUpdate, requestWrapper.Message.Body)
	if err != nil {
		return
	}

	id := requestWrapper.Message.Res[strings.LastIndex(requestWrapper.Message.Res, "/") + 1:]
	response.Body, finalInterceptorBody, err = database.Adapter.Update(a.class, id, requestWrapper.Message.Body)
	return
}

var handleDelete = func(a *Actor, requestWrapper messages.RequestWrapper) (response messages.Message, err *utils.Error) {

	if strings.EqualFold(a.actorType, constants.ActorTypeModel) {
		// delete object
		id := requestWrapper.Message.Res[strings.LastIndex(requestWrapper.Message.Res, "/") + 1:]
		response.Body, err = database.Adapter.Delete(a.class, id)
		if err == nil {
			response.Status = http.StatusNoContent
		}
	}
	return
}

func getChildRes(res, parentRes string) (fullPath string) {
	res = strings.Trim(res, "/")
	parentRes = strings.Trim(parentRes, "/")
	currentResSize := len(parentRes)
	resSuffix := res[currentResSize:]
	trimmedSuffix := strings.Trim(resSuffix, "/")
	directChild := strings.Split(trimmedSuffix, "/")
	relativePath := directChild[0]
	if len(parentRes) > 0 {
		fullPath = "/" + parentRes + "/" + relativePath
	} else {
		fullPath = "/" + relativePath
	}
	return
}

func (a *Actor) checkAndSend(c chan messages.Message, m messages.Message) {
	defer func() {
		if r := recover(); r != nil {
			log.Error(a.res + "Trying to send on closed channel.")
		}
	}()
	c <- m
}

var copyMessage = func(message messages.Message) (copy messages.Message) {
	copy.Res = message.Res
	copy.Command = message.Command
	copy.Headers = message.Headers
	copy.Parameters = message.Parameters
	copy.Body = message.Body
	copy.RawBody = message.RawBody
	return
}