package core

import (
	"io"
	"strings"
	"net/http"
	"encoding/json"
	"github.com/rihtim/core/log"
	"github.com/rihtim/core/utils"
	"github.com/rihtim/core/functions"
	"github.com/rihtim/core/messages"
	"github.com/rihtim/core/constants"
	"github.com/rihtim/core/requestscope"
	"github.com/rihtim/core/interceptors"
	"github.com/rihtim/core/requesthandler"
	//"github.com/Sirupsen/logrus"
	//"runtime/debug"
	"github.com/getsentry/raven-go"
)

var Port = "1707"
var BodyParserExcludedResources map[string]bool

var AddFunctionHandler = func(path string, handler functions.FunctionHandler) {
	functions.AddFunctionHandler(path, handler)
}

var AddInterceptor = func(res, method string, interceptorType interceptors.InterceptorType, interceptor interceptors.Interceptor) {
	interceptors.AddInterceptor(res, method, interceptorType, interceptor, nil)
}

var AddInterceptorWithExtras = func(res, method string, interceptorType interceptors.InterceptorType, interceptor interceptors.Interceptor, extras interface{}) {
	interceptors.AddInterceptor(res, method, interceptorType, interceptor, extras)
}

var wrapInRaven = false

var Serve = func() {

	/* if ravenConfig, containsRavenConfig := Configuration["raven"].(map[string]interface{}); containsRavenConfig {
		raven.SetDSN(ravenConfig["url"].(string))
		wrapInRaven = true
		log.Info("Initialising raven.")
	} */

	log.Info("Starting server on port " + Port + ".")

	http.HandleFunc("/", handler)
	serveErr := http.ListenAndServe(":"+Port, nil)
	if serveErr != nil {
		log.Info("Serving on port " + Port + " failed. Be sure that the port is available.")
	}
}

func handler(w http.ResponseWriter, r *http.Request) {

	// parsing request
	request, parseReqErr := parseRequest(r)
	if parseReqErr != nil {
		printError(w, parseReqErr)
		return
	}

	requestScope := requestscope.Init()

	if (wrapInRaven) {
		raven.CapturePanic(func() {
			response, _, err := HandleRequest(request, requestScope)
			buildResponse(w, response, err)
		}, nil)
	} else {
		response, _, err := HandleRequest(request, requestScope)
		buildResponse(w, response, err)
	}
}

var HandleRequest = func(request messages.Message, requestScope requestscope.RequestScope) (response messages.Message, updatedRequestScope requestscope.RequestScope, err *utils.Error) {

	/*defer func() {
		if err := recover(); err != nil {
			log.WithFields(logrus.Fields{
				"error ": err,
				"stackTrace": string(debug.Stack()),
			}).Error("Crash recovered!")
		}
	}()*/

	var editedRequest, editedResponse messages.Message
	var editedRequestScope requestscope.RequestScope

	// execute BEFORE_EXEC interceptors
	editedRequest, editedResponse, editedRequestScope, err = interceptors.ExecuteInterceptors(request.Res, request.Command, interceptors.BEFORE_EXEC, requestScope, request, response)
	if err != nil {
		response, err = handleError(request, editedResponse, requestScope, err)
		return
	}

	if !editedResponse.IsEmpty() {
		response = editedResponse
		return
	}

	// update request if interceptor returned an edited request
	if !editedRequest.IsEmpty() {
		request = editedRequest
	}

	// update request scope if interceptor returned an editedRequestScope
	if !editedRequestScope.IsEmpty() {
		requestScope = editedRequestScope
	}

	// execute the request
	if functions.ContainsHandler(request.Res) {
		response, editedRequestScope, err = functions.ExecuteFunction(request, requestScope)
	} else {
		response, editedRequestScope, err = requesthandler.HandleRequest(request, requestScope)
	}

	if err != nil {
		response, err = handleError(request, editedResponse, requestScope, err)
		return
	}

	// update request scope if interceptor returned an editedRequestScope
	if !editedRequestScope.IsEmpty() {
		requestScope = editedRequestScope
	}

	// execute AFTER_EXEC interceptors
	_, editedResponse, editedRequestScope, err = interceptors.ExecuteInterceptors(request.Res, request.Command, interceptors.AFTER_EXEC, requestScope, request, response)

	// update response if interceptor returned an edited response
	if !editedResponse.IsEmpty() {
		response = editedResponse
	}

	// update request scope if interceptor returned an editedRequestScope
	if !editedRequestScope.IsEmpty() {
		requestScope = editedRequestScope
	}

	// execute FINAL interceptors in goroutine
	go interceptors.ExecuteInterceptors(request.Res, request.Command, interceptors.FINAL, requestScope, request, response)

	return
}

func handleError(request, response messages.Message, requestScope requestscope.RequestScope, err *utils.Error) (returnedResponse messages.Message, returnedErr *utils.Error) {

	returnedErr = err
	returnedResponse = response

	requestScope.Set("error", err)

	var editedResponse messages.Message
	_, editedResponse, _, err = interceptors.ExecuteInterceptors(request.Res, request.Command, interceptors.ON_ERROR, requestScope, request, response)

	if err != nil {
		returnedErr = err
	}
	if !editedResponse.IsEmpty() {
		returnedResponse = editedResponse
	}
	return
}

func printError(w http.ResponseWriter, err *utils.Error) {
	bytes, cbErr := json.Marshal(map[string]string{"message": err.Message})
	if cbErr != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		log.Error("Generating error message failed.")
	}
	log.Error(err.Message)
	w.WriteHeader(err.Code)
	io.WriteString(w, string(bytes))
}

/*func handler_old(w http.ResponseWriter, r *http.Request) {

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	if origin := r.Header.Get("Origin"); origin != "" {
		w.Header().Set("Access-Control-Allow-Origin", origin)
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
		w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, Session-Token")
	}
	// Stop here if its Pre-flighted OPTIONS request
	if r.Method == "OPTIONS" {
		return
	}

	requestWrapper, parseReqErr := parseRequest(r)
	if parseReqErr != nil {
		bytes, err := json.Marshal(map[string]string{"message":parseReqErr.Message})
		if err != nil {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			log.Error("Generating parse error message failed.")
		}
		log.Error(parseReqErr.Message)
		w.WriteHeader(parseReqErr.Code)
		io.WriteString(w, string(bytes))
		return
	}

	//	responseChannel := make(chan messages.Message)
	//	requestWrapper.Listener = responseChannel

	//	log.Debug("Sending request actor")
	actor := actors.CreateActorForRes(requestWrapper.Message.Res)
	response, err := actors.HandleRequest(&actor, requestWrapper)

	for k, v := range response.Headers {
		//vAsArray := v.([]string)
		w.Header().Set(k, v[0])
	}

	if err != nil {
		if response.Status == 0 {response.Status = err.Code}
		if response.Body == nil {response.Body = map[string]interface{}{"code":err.Code, "message":err.Message}}
		log.WithFields(logrus.Fields{
			"res": requestWrapper.Message.Res,
			"command": requestWrapper.Message.Command,
			"errorCode": err.Code,
			"errorMessage": err.Message,
		}).Error("Got error.")
	}
	//	RootActor.Inbox <- requestWrapper
	//	response := <-responseChannel


	if response.Status != 0 {
		w.WriteHeader(response.Status)
	}

	if response.RawBody != nil {
		//w.Header().Set("Content-Type", "text/plain")
		w.Write(response.RawBody)
	}

	if response.Body != nil {
		bytes, encodeErr := json.Marshal(response.Body)
		if encodeErr != nil {
			log.Error("Encoding response body failed.")
			http.Error(w, "Internal server error", http.StatusInternalServerError)
		}
		io.WriteString(w, string(bytes))
	}
	//	close(responseChannel)
}*/

func parseRequest(r *http.Request) (request messages.Message, err *utils.Error) {

	res := strings.TrimRight(r.URL.Path, "/")
	if strings.EqualFold(res, "") {
		err = &utils.Error{http.StatusBadRequest, "Root path '/' cannot be requested directly. " }
		return
	}
	request = messages.Message{
		Res:        res,
		Command:    strings.ToLower(r.Method),
		Headers:    r.Header,
		Parameters: r.URL.Query(),
	}

	if strings.Index(res, constants.ResourceTypeFiles) == 0 {
		if r.Body == nil {
			err = &utils.Error{http.StatusBadRequest, "Request body cannot be empty for create file requests."}
			return
		}
		request.ReqBodyRaw = r.Body
	} else {

		if BodyParserExcludedResources != nil && BodyParserExcludedResources[res] {
			request.ReqBodyRaw = r.Body
			return
		}
		readErr := json.NewDecoder(r.Body).Decode(&request.Body)
		if readErr != nil && readErr != io.EOF {
			err = &utils.Error{Code: http.StatusBadRequest, Message: "Parsing request body failed. Reason: " + readErr.Error()}
			return
		}
	}

	return
}

func buildResponse(w http.ResponseWriter, response messages.Message, err *utils.Error) {

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	for k, v := range response.Headers {
		w.Header().Set(k, v[0])
	}

	if err != nil {
		if response.Status == 0 {
			response.Status = err.Code
		}
		if response.Body == nil {
			response.Body = map[string]interface{}{"code": err.Code, "message": err.Message}
		}
	}

	if response.Status != 0 {
		w.WriteHeader(response.Status)
	}

	if response.RawBody != nil {
		w.Write(response.RawBody)
	}

	if response.Body != nil {
		bytes, encodeErr := json.Marshal(response.Body)
		if encodeErr != nil {
			log.Error("Encoding response body failed.")
			http.Error(w, "Internal server error", http.StatusInternalServerError)
		}
		io.WriteString(w, string(bytes))
	}
}
