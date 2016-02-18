package core

import (
	"io"
	"os"
	"strings"
	"net/http"
	"encoding/json"
	"github.com/rihtim/core/utils"
	"github.com/rihtim/core/config"
	"github.com/rihtim/core/functions"
	"github.com/rihtim/core/interceptors"
	"github.com/rihtim/core/database"
	"github.com/rihtim/core/messages"
	"github.com/rihtim/core/actors"
	"github.com/rihtim/core/constants"
	log "github.com/Sirupsen/logrus"
)

var Configuration map[string]interface{}
var RootActor actors.Actor

var InitWithConfig = func(fileName string) (err *utils.Error) {

	log.Info("Initialising with config file: " + fileName)

	// reading and parsing configuration
	Configuration, err = config.ReadConfig(fileName)
	if err != nil {
		log.Fatal(err.Message)
		return
	}

	// initialising database adapter
	database.Adapter = new(database.MongoAdapter)
	dbInitErr := database.Adapter.Init(Configuration["mongo"].(map[string]interface{}))
	if dbInitErr != nil {
		log.Fatal(dbInitErr.Message)
		os.Exit(dbInitErr.Code)
	}

	// connecting to the database
	dbConnErr := database.Adapter.Connect()
	if dbConnErr != nil {
		log.Fatal(dbConnErr.Message)
		os.Exit(dbConnErr.Code)
	}

	// creating root actor
	RootActor = actors.CreateActor(nil, "/")
	go RootActor.Run()

	return
}

var AddFunctionHandler = func(path string, handler functions.FunctionHandler) {
	functions.AddFunctionHandler(path, handler)
}

var AddInterceptor = func(res, method string, interceptorType interceptors.InterceptorType, interceptor interceptors.Interceptor) {
	interceptors.AddInterceptor(res, method, interceptorType, interceptor)
}

var Serve = func() {

	port := Configuration["port"].(string)
	log.Info("Starting server on port " + port + ".")

	http.HandleFunc("/", handler)
	serveErr := http.ListenAndServe(":" + port, nil)
	if serveErr != nil {
		log.Info("Serving on port " + port + " failed. Be sure that the port is available.")
	}
}

func handler(w http.ResponseWriter, r *http.Request) {

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	if origin := r.Header.Get("Origin"); origin != "" {
		w.Header().Set("Access-Control-Allow-Origin", origin)
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
		w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")
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

	responseChannel := make(chan messages.Message)
	requestWrapper.Listener = responseChannel

	RootActor.Inbox <- requestWrapper
	response := <-responseChannel
	if response.Status != 0 {
		w.WriteHeader(response.Status)
	}

	if response.RawBody != nil {
		w.Header().Set("Content-Type", "text/plain")
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
	close(responseChannel)
}


func parseRequest(r *http.Request) (requestWrapper messages.RequestWrapper, err *utils.Error) {

	res := strings.TrimRight(r.URL.Path, "/")
	if strings.EqualFold(res, "") {
		err = &utils.Error{http.StatusBadRequest, "Root path '/' cannot be requested directly. " }
		return
	}
	requestWrapper.Res = res
	requestWrapper.Message.Res = res
	requestWrapper.Message.Command = strings.ToLower(r.Method)
	requestWrapper.Message.Headers = r.Header
	requestWrapper.Message.Parameters = r.URL.Query()

	if strings.Index(res, constants.ResourceTypeFiles) == 0 {
		if r.Body == nil {
			err = &utils.Error{http.StatusBadRequest, "Request body cannot be empty for create file requests."}
			return
		}
		requestWrapper.Message.ReqBodyRaw = r.Body
	} else {
		readErr := json.NewDecoder(r.Body).Decode(&requestWrapper.Message.Body)
		if readErr != nil && readErr != io.EOF {
			err = &utils.Error{http.StatusBadRequest, "Request body is not a valid json."}
			return
		}
	}

	return
}

