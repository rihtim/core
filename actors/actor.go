package actors

import (
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
	"github.com/rihtim/core/modifier"
	"github.com/rihtim/core/log"

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

var CreateActor = func(parent *Actor, res string) (a Actor) {
	log.Debug("Creating actor for " + res)

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
	if parent != nil {a.parentInbox = parent.Inbox}
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

			if requestWrapper.Res == a.res {
				// if the resource of the message is this actor's resource

				messageString, mErr := json.Marshal(requestWrapper.Message)
				if mErr == nil {
					log.Info(a.res + ": Handling " + string(messageString))
				}

				response, err := handleRequest(a, requestWrapper)

				if err != nil {
					if response.Status == 0 {response.Status = err.Code}
					if response.Body == nil {response.Body = map[string]interface{}{"message":err.Message}}
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
				childRes := getChildRes(requestWrapper.Res, a.res)

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

var handleRequest = func(a *Actor, requestWrapper messages.RequestWrapper) (response messages.Message, err *utils.Error) {

	// check for method is allowed on the resource type
	allowedMethods := AllowedMethodsOfActorTypes[a.actorType]
	isMethodAllowed := allowedMethods[strings.ToLower(requestWrapper.Message.Command)]
	if !isMethodAllowed {
		err = &utils.Error{http.StatusMethodNotAllowed, http.StatusText(http.StatusMethodNotAllowed)}
		return
	}
	message := requestWrapper.Message
	var finalInterceptorBody map[string]interface{}

	// call interceptors before authentication
	message, err = interceptors.ExecuteInterceptors(message.Res, message.Command, interceptors.BEFORE_AUTH, nil, message)
	if err != nil {
		return
	}

	// check permissions of user on this resource. continue anyway if the actor type is ActorTypeFunctions
	isGranted, user, authErr := auth.IsGranted(a.class, requestWrapper)
	if !isGranted && !strings.EqualFold(a.actorType, constants.ActorTypeFunctions) {
		if authErr != nil {
			err = authErr
		} else {
			err = &utils.Error{http.StatusUnauthorized, http.StatusText(http.StatusUnauthorized)}
		}
		return
	}

	// call interceptors before execution
	message, err = interceptors.ExecuteInterceptors(message.Res, message.Command, interceptors.BEFORE_EXEC, user, message)
	if err != nil {
		return
	}

	if (strings.EqualFold(a.actorType, constants.ActorTypeFunctions)) {
		functionHandler := functions.GetFunctionHandler(message.Res)
		response, finalInterceptorBody, err = functionHandler(user, message)
	} else if strings.EqualFold(requestWrapper.Message.Command, constants.CommandPost) {
		response, finalInterceptorBody, err = handlePost(a, requestWrapper, user)
	} else if strings.EqualFold(requestWrapper.Message.Command, constants.CommandGet) {
		response, err = handleGet(a, requestWrapper)
	} else if strings.EqualFold(requestWrapper.Message.Command, constants.CommandPut) {
		response, finalInterceptorBody, err = handlePut(a, requestWrapper)
	} else if strings.EqualFold(requestWrapper.Message.Command, constants.CommandDelete) {
		response, err = handleDelete(a, requestWrapper)
	}
	if err != nil {
		return
	}

	// call interceptors after execution
	message, err = interceptors.ExecuteInterceptors(message.Res, message.Command, interceptors.AFTER_EXEC, user, message)
	if err != nil {
		return
	}

	// call interceptors on final
	finalMessage := copyMessage(response)
	finalMessage.Body = finalInterceptorBody
	go interceptors.ExecuteInterceptors(message.Res, message.Command, interceptors.FINAL, user, finalMessage)
	return
}

var handlePost = func(a *Actor, requestWrapper messages.RequestWrapper, user interface{}) (response messages.Message, hookBody map[string]interface{}, err *utils.Error) {

	if (!strings.EqualFold(a.class, constants.ClassFiles)) {
		response.Body, hookBody, err = database.Adapter.Create(a.class, requestWrapper.Message.Body)
	} else {
		response.Body, hookBody, err = database.Adapter.CreateFile(requestWrapper.Message.ReqBodyRaw)
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
	if requestWrapper.Message.Parameters["expand"] != nil {
		expandConfig := requestWrapper.Message.Parameters["expand"][0]
		if _, hasDataArray := response.Body["results"]; hasDataArray {
			response.Body, err = modifier.ExpandArray(response.Body, expandConfig)
		} else {
			response.Body, err = modifier.ExpandItem(response.Body, expandConfig)
		}
	}
	return
}

var handlePut = func(a *Actor, requestWrapper messages.RequestWrapper) (response messages.Message, hookBody map[string]interface{}, err *utils.Error) {

	if strings.EqualFold(a.actorType, constants.ActorTypeModel) {
		// update object
		id := requestWrapper.Message.Res[strings.LastIndex(requestWrapper.Message.Res, "/") + 1:]
		response.Body, hookBody, err = database.Adapter.Update(a.class, id, requestWrapper.Message.Body)
	}
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