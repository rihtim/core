package core

import (
	"io"
	"strings"
	"net/http"
	"encoding/json"

	"github.com/rihtim/core/log"
	"github.com/rihtim/core/utils"
	"github.com/rihtim/core/messages"
	"github.com/rihtim/core/requestscope"
	"github.com/rihtim/core/interceptors"
	"github.com/rihtim/core/functions"
	"github.com/rihtim/core/dataprovider"
)

var Functions functions.FunctionController = &functions.CoreFunctionController{}
var Interceptors interceptors.InterceptorController = &interceptors.CoreInterceptorController{}
var DataProvider dataprovider.Provider

var BodyParserExcludedPaths map[string]bool

func HandleHttpRequest(w http.ResponseWriter, r *http.Request) {

	// parse request
	request, parseReqErr := parseRequest(r)
	if parseReqErr != nil {
		printError(w, parseReqErr)
		return
	}

	response, _, err := HandleRequest(request, requestscope.Init())
	buildResponse(w, response, err)
}

func HandleRequest(request messages.Message, requestScope requestscope.RequestScope) (response messages.Message, updatedRequestScope requestscope.RequestScope, err *utils.Error) {

	var editedRequest, editedResponse messages.Message
	var editedRequestScope requestscope.RequestScope

	// execute BEFORE_EXEC interceptors
	editedRequest, editedResponse, editedRequestScope, err = Interceptors.Execute(request.Res, request.Command, interceptors.BEFORE_EXEC, requestScope, request, response, DataProvider)
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
	if Functions.Contains(request.Res, request.Command) {
		response, editedRequestScope, err = Functions.Execute(request, requestScope, DataProvider)
	} else {
		response, editedRequestScope, err = Execute(request, DataProvider)
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
	_, editedResponse, editedRequestScope, err = Interceptors.Execute(request.Res, request.Command, interceptors.AFTER_EXEC, requestScope, request, response, DataProvider)

	// update response if interceptor returned an edited response
	if !editedResponse.IsEmpty() {
		response = editedResponse
	}

	// update request scope if interceptor returned an editedRequestScope
	if !editedRequestScope.IsEmpty() {
		requestScope = editedRequestScope
	}

	// execute FINAL interceptors in goroutine
	go Interceptors.Execute(request.Res, request.Command, interceptors.FINAL, requestScope, request, response, DataProvider)

	return
}

func handleError(request, response messages.Message, requestScope requestscope.RequestScope, err *utils.Error) (returnedResponse messages.Message, returnedErr *utils.Error) {

	returnedErr = err
	returnedResponse = response

	requestScope.Set("error", err)

	var editedResponse messages.Message
	_, editedResponse, _, err = Interceptors.Execute(request.Res, request.Command, interceptors.ON_ERROR, requestScope, request, response, DataProvider)

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

func parseRequest(r *http.Request) (request messages.Message, err *utils.Error) {

	res := strings.TrimRight(r.URL.Path, "/")
	if strings.EqualFold(res, "") {
		err = &utils.Error{
			Code:    http.StatusBadRequest,
			Message: "Root path '/' cannot be requested directly. ",
		}
		return
	}

	request = messages.Message{
		IP:         getIPAdress(r),
		Res:        res,
		Command:    strings.ToLower(r.Method),
		Headers:    r.Header,
		Parameters: r.URL.Query(),
	}
	request.ReqBodyRaw = r.Body

	// return if the requests for this path are excluded for parsing
	if BodyParserExcludedPaths != nil && BodyParserExcludedPaths[res] {
		return
	}

	readErr := json.NewDecoder(r.Body).Decode(&request.Body)
	if readErr != nil && readErr != io.EOF {
		err = &utils.Error{Code: http.StatusBadRequest, Message: "Parsing request body failed. Reason: " + readErr.Error()}
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
