package auth

import (
	"testing"
	. "github.com/smartystreets/goconvey/convey"
	"github.com/eluleci/dock/adapters"
	"github.com/eluleci/dock/messages"
	"net/http"
	"github.com/eluleci/dock/utils"
	"net/http/httptest"
	"net/url"
	"fmt"
	"github.com/eluleci/dock/config"
	"strings"
)

func setDefaultServer(mockServer *httptest.Server) {

	// transport reroutes all traffic to the example server
	transport := &http.Transport{
		Proxy: func(req *http.Request) (*url.URL, error) {
			return url.Parse(mockServer.URL)
		},
	}

	// replacing real http client
	httpClient = &http.Client{Transport: transport}
}

func makeValidFacebookRequest() messages.RequestWrapper {

	facebookData := make(map[string]interface{})
	facebookData["id"] = "facebookuserid"
	facebookData["accessToken"] = "facebookaccesstoken"

	var message messages.Message
	message.Body = make(map[string]interface{})
	message.Body["facebook"] = facebookData

	var requestWrapper messages.RequestWrapper
	requestWrapper.Message = message

	return requestWrapper
}

func makeValidGoogleRequest() messages.RequestWrapper {

	googleData := make(map[string]interface{})
	googleData["id"] = "someid"
	googleData["idToken"] = "someidtoken"

	var message messages.Message
	message.Body = make(map[string]interface{})
	message.Body["google"] = googleData

	var requestWrapper messages.RequestWrapper
	requestWrapper.Message = message

	return requestWrapper
}

func TestHandleSignUp(t *testing.T) {

	Convey("Should return bad request", t, func() {

		var called bool
		getAccountData = func(requestWrapper messages.RequestWrapper, dbAdapter *adapters.MongoAdapter) (accountData map[string]interface{}, err *utils.Error) {
			called = true
			return
		}

		var message messages.Message

		var requestWrapper messages.RequestWrapper
		requestWrapper.Message = message

		_, _, err := HandleSignUp(requestWrapper, &adapters.MongoAdapter{})

		So(err.Code, ShouldEqual, http.StatusBadRequest)
		So(called, ShouldBeFalse)
	})

	Convey("Should return bad request for password", t, func() {

		var called bool
		getAccountData = func(requestWrapper messages.RequestWrapper, dbAdapter *adapters.MongoAdapter) (accountData map[string]interface{}, err *utils.Error) {
			called = true
			return
		}

		var message messages.Message
		message.Body = make(map[string]interface{})
		message.Body["username"] = "elgefe"

		var requestWrapper messages.RequestWrapper
		requestWrapper.Message = message

		_, _, err := HandleSignUp(requestWrapper, &adapters.MongoAdapter{})

		So(err.Code, ShouldEqual, http.StatusBadRequest)
		So(called, ShouldBeFalse)
	})

	Convey("Should return conflict", t, func() {

		getAccountData = func(requestWrapper messages.RequestWrapper, dbAdapter *adapters.MongoAdapter) (accountData map[string]interface{}, err *utils.Error) {
			accountData = make(map[string]interface{})
			return
		}

		var called bool
		generateToken = func(userId string, userData map[string]interface{}) (tokenString string, err *utils.Error) {
			called = true
			err = &utils.Error{http.StatusConflict, "Exists."}
			return
		}

		var message messages.Message
		message.Body = make(map[string]interface{})
		message.Body["email"] = "email@domain.com"
		message.Body["password"] = "apasswordimpossibletofind"

		var requestWrapper messages.RequestWrapper
		requestWrapper.Message = message

		_, _, err := HandleSignUp(requestWrapper, &adapters.MongoAdapter{})

		So(err.Code, ShouldEqual, http.StatusConflict)
		So(called, ShouldBeFalse)
	})

	Convey("Should return internal server error", t, func() {

		getAccountData = func(requestWrapper messages.RequestWrapper, dbAdapter *adapters.MongoAdapter) (accountData map[string]interface{}, err *utils.Error) {
			return
		}

		generateToken = func(userId string, userData map[string]interface{}) (tokenString string, err *utils.Error) {
			err = &utils.Error{http.StatusInternalServerError, "Generating token failed."}
			return
		}

		adapters.Create = func(collection string, data map[string]interface{}) (response map[string]interface{}, hookBody map[string]interface{}, err *utils.Error) {
			response = make(map[string]interface{})
			response["_id"] = "564f1a28e63bce219e1cc745"
			return
		}

		var message messages.Message
		message.Body = make(map[string]interface{})
		message.Body["email"] = "email@domain.com"
		message.Body["password"] = "apasswordimpossibletofind"

		var requestWrapper messages.RequestWrapper
		requestWrapper.Message = message

		response, _, _ := HandleSignUp(requestWrapper, &adapters.MongoAdapter{})

		So(response.Status, ShouldEqual, http.StatusInternalServerError)
	})

	Convey("Should call auth.getAccountData with email", t, func() {

		var called bool
		getAccountData = func(requestWrapper messages.RequestWrapper, dbAdapter *adapters.MongoAdapter) (accountData map[string]interface{}, err *utils.Error) {
			called = true
			return
		}

		adapters.Create = func(collection string, data map[string]interface{}) (response map[string]interface{}, hookBody map[string]interface{}, err *utils.Error) {
			response = make(map[string]interface{})
			response["_id"] = "564f1a28e63bce219e1cc745"
			return
		}

		var message messages.Message
		message.Body = make(map[string]interface{})
		message.Body["email"] = "email@domain.com"
		message.Body["password"] = "apasswordimpossibletofind"

		var requestWrapper messages.RequestWrapper
		requestWrapper.Message = message

		HandleSignUp(requestWrapper, &adapters.MongoAdapter{})

		So(called, ShouldBeTrue)
	})

	Convey("Should call auth.getAccountData with username", t, func() {

		var called bool
		getAccountData = func(requestWrapper messages.RequestWrapper, dbAdapter *adapters.MongoAdapter) (accountData map[string]interface{}, err *utils.Error) {
			called = true
			return
		}

		adapters.Create = func(collection string, data map[string]interface{}) (response map[string]interface{}, hookBody map[string]interface{}, err *utils.Error) {
			response = make(map[string]interface{})
			response["_id"] = "564f1a28e63bce219e1cc745"
			return
		}

		var message messages.Message
		message.Body = make(map[string]interface{})
		message.Body["username"] = "lordoftherings"
		message.Body["password"] = "apasswordimpossibletofind"

		var requestWrapper messages.RequestWrapper
		requestWrapper.Message = message

		HandleSignUp(requestWrapper, &adapters.MongoAdapter{})

		So(called, ShouldBeTrue)
	})

	Convey("Should create account", t, func() {

		getAccountData = func(requestWrapper messages.RequestWrapper, dbAdapter *adapters.MongoAdapter) (accountData map[string]interface{}, err *utils.Error) {
			return
		}

		generateToken = func(userId string, userData map[string]interface{}) (tokenString string, err *utils.Error) {
			tokenString = ""
			return
		}

		adapters.Create = func(collection string, data map[string]interface{}) (response map[string]interface{}, hookBody map[string]interface{}, err *utils.Error) {
			response = make(map[string]interface{})
			response["_id"] = "564f1a28e63bce219e1cc745"
			return
		}

		var message messages.Message
		message.Body = make(map[string]interface{})
		message.Body["email"] = "email@domain.com"
		message.Body["password"] = "apasswordimpossibletofind"

		var requestWrapper messages.RequestWrapper
		requestWrapper.Message = message

		response, _, _ := HandleSignUp(requestWrapper, &adapters.MongoAdapter{})

		So(response.Status, ShouldEqual, http.StatusCreated)
	})
}

func TestFacebookRegistration(t *testing.T) {

	// https needs to be changed to http for mock server to work
	facebookTokenVerificationEndpoint = "http://graph.facebook.com/debug_token"

	config.SystemConfig = config.Config{}
	config.SystemConfig.Facebook = map[string]string{
		"appId": "facebookappid",
		"appToken": "facebookapptoken",
	}

	Convey("Should fail creating account with Facebook when id is missing", t, func() {

		facebookData := make(map[string]interface{})
		facebookData["accessToken"] = "facebookaccesstoken"

		var message messages.Message
		message.Body = make(map[string]interface{})
		message.Body["facebook"] = facebookData

		var requestWrapper messages.RequestWrapper
		requestWrapper.Message = message

		_, _, err := HandleSignUp(requestWrapper, &adapters.MongoAdapter{})

		So(err.Code, ShouldEqual, http.StatusBadRequest)
	})

	Convey("Should fail creating account with Facebook when access token is missing", t, func() {

		facebookData := make(map[string]interface{})
		facebookData["id"] = "facebookuserid"

		var message messages.Message
		message.Body = make(map[string]interface{})
		message.Body["facebook"] = facebookData

		var requestWrapper messages.RequestWrapper
		requestWrapper.Message = message

		_, _, err := HandleSignUp(requestWrapper, &adapters.MongoAdapter{})

		So(err.Code, ShouldEqual, http.StatusBadRequest)
	})

	Convey("Should fail creating account with Facebook when server configuration is missing", t, func() {

		// removing google configuration
		originalConfig := config.SystemConfig
		config.SystemConfig.Facebook = nil

		requestWrapper := makeValidFacebookRequest()
		_, _, err := HandleSignUp(requestWrapper, &adapters.MongoAdapter{})

		So(err.Code, ShouldEqual, http.StatusInternalServerError)

		// revert back the original configuration
		config.SystemConfig = originalConfig
	})

	Convey("Should fail when getting response from Facebook fails", t, func() {

		mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(500)
		}))
		defer mockServer.Close()
		setDefaultServer(mockServer)

		requestWrapper := makeValidFacebookRequest()
		_, _, err := HandleSignUp(requestWrapper, &adapters.MongoAdapter{})

		So(err.Code, ShouldEqual, http.StatusInternalServerError)
	})

	Convey("Should fail when parsing response from Facebook fails", t, func() {

		mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(200)
			fmt.Fprintln(w, `{invalidjsonresponse`)
		}))
		defer mockServer.Close()
		setDefaultServer(mockServer)

		requestWrapper := makeValidFacebookRequest()
		_, _, err := HandleSignUp(requestWrapper, &adapters.MongoAdapter{})

		So(err.Code, ShouldEqual, http.StatusInternalServerError)
	})

	Convey("Should fail when Facebook's response is not as expected", t, func() {

		requestWrapper := makeValidFacebookRequest()

		// when 'data' is not at root
		mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(200)
			fmt.Fprintln(w, `{}`)
		}))
		defer mockServer.Close()
		setDefaultServer(mockServer)

		_, _, err := HandleSignUp(requestWrapper, &adapters.MongoAdapter{})

		So(err.Code, ShouldEqual, http.StatusInternalServerError)

		// when required fields are not inside 'data'
		mockServer2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(200)
			fmt.Fprintln(w, `{"data": {}}`)
		}))
		defer mockServer2.Close()
		setDefaultServer(mockServer2)

		_, _, err2 := HandleSignUp(requestWrapper, &adapters.MongoAdapter{})

		So(err2.Code, ShouldEqual, http.StatusInternalServerError)
	})

	Convey("Should fail when app ids don't match.", t, func() {

		mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(200)
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprintln(w, `{"data": {"app_id": "invalidfacebookappid", "user_id": "facebookuserid", "is_valid": true}}`)
		}))
		defer mockServer.Close()
		setDefaultServer(mockServer)

		requestWrapper := makeValidFacebookRequest()

		_, _, err := HandleSignUp(requestWrapper, &adapters.MongoAdapter{})

		So(err.Code, ShouldEqual, http.StatusInternalServerError)
	})

	Convey("Should fail when user ids don't match.", t, func() {

		mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(200)
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprintln(w, `{"data": {"app_id": "facebookappid", "user_id": "invalidfacebookuserid", "is_valid": true}}`)
		}))
		defer mockServer.Close()
		setDefaultServer(mockServer)

		requestWrapper := makeValidFacebookRequest()

		_, _, err := HandleSignUp(requestWrapper, &adapters.MongoAdapter{})

		So(err.Code, ShouldEqual, http.StatusBadRequest)
	})

	Convey("Should fail when token is not valid.", t, func() {

		mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(200)
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprintln(w, `{"data": {"app_id": "facebookappid", "user_id": "facebookuserid", "is_valid": false}}`)
		}))
		defer mockServer.Close()
		setDefaultServer(mockServer)

		requestWrapper := makeValidFacebookRequest()

		_, _, err := HandleSignUp(requestWrapper, &adapters.MongoAdapter{})

		So(err.Code, ShouldEqual, http.StatusBadRequest)
	})

	Convey("Should create new account.", t, func() {

		mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(200)
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprintln(w, `{"data": {"app_id": "facebookappid", "user_id": "facebookuserid", "is_valid": true}}`)
		}))
		defer mockServer.Close()
		setDefaultServer(mockServer)

		var called bool
		getAccountData = func(requestWrapper messages.RequestWrapper, dbAdapter *adapters.MongoAdapter) (accountData map[string]interface{}, err *utils.Error) {
			called = true
			// returning nil account info
			return
		}

		requestWrapper := makeValidFacebookRequest()
		response, _, err := HandleSignUp(requestWrapper, &adapters.MongoAdapter{})

		So(err, ShouldBeNil)
		So(called, ShouldBeTrue)
		So(response.Status, ShouldEqual, http.StatusCreated)
	})

	Convey("Should return existing account.", t, func() {

		mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(200)
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprintln(w, `{"data": {"app_id": "facebookappid", "user_id": "facebookuserid", "is_valid": true}}`)
		}))
		defer mockServer.Close()
		setDefaultServer(mockServer)

		var called bool
		getAccountData = func(requestWrapper messages.RequestWrapper, dbAdapter *adapters.MongoAdapter) (accountData map[string]interface{}, err *utils.Error) {
			called = true
			accountData = make(map[string]interface{})
			accountData["_id"] = "564f1a28e63bce219e1cc745"
			// returning existing account
			return
		}

		requestWrapper := makeValidFacebookRequest()
		response, _, err := HandleSignUp(requestWrapper, &adapters.MongoAdapter{})

		So(err, ShouldBeNil)
		So(called, ShouldBeTrue)
		So(response.Status, ShouldEqual, http.StatusCreated)
	})
}

func TestGoogleRegistration(t *testing.T) {

	// https needs to be changed to http for mock server to work
	googleTokenVerificationEndpoint = "http://mock.google.com"

	// setting configuration to default
	config.SystemConfig = config.Config{}
	config.SystemConfig.Google = map[string]string {
		"clientId": "googleclientid",
	}

	Convey("Should fail creating account with Google when idToken is missing", t, func() {

		googleData := make(map[string]interface{})
		googleData["id"] = "someid"

		var message messages.Message
		message.Body = make(map[string]interface{})
		message.Body["google"] = googleData

		var requestWrapper messages.RequestWrapper
		requestWrapper.Message = message

		_, _, err := HandleSignUp(requestWrapper, &adapters.MongoAdapter{})

		So(err.Code, ShouldEqual, http.StatusBadRequest)
	})

	Convey("Should fail creating account with Google when id is missing", t, func() {

		googleData := make(map[string]interface{})
		googleData["idToken"] = "someidtoken"

		var message messages.Message
		message.Body = make(map[string]interface{})
		message.Body["google"] = googleData

		var requestWrapper messages.RequestWrapper
		requestWrapper.Message = message

		_, _, err := HandleSignUp(requestWrapper, &adapters.MongoAdapter{})

		So(err.Code, ShouldEqual, http.StatusBadRequest)
	})

	Convey("Should fail creating account with Google when server configuration is missing", t, func() {

		// removing google configuration
		correctConfig := config.SystemConfig
		config.SystemConfig.Google = nil

		requestWrapper := makeValidGoogleRequest()

		_, _, err := HandleSignUp(requestWrapper, &adapters.MongoAdapter{})

		So(err.Code, ShouldEqual, http.StatusInternalServerError)

		// revert back the google configuration
		config.SystemConfig = correctConfig
	})

	Convey("Should fail when getting response from Google fails", t, func() {

		mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(500)
		}))
		defer mockServer.Close()
		setDefaultServer(mockServer)

		requestWrapper := makeValidGoogleRequest()

		_, _, err := HandleSignUp(requestWrapper, &adapters.MongoAdapter{})

		So(err.Code, ShouldEqual, http.StatusInternalServerError)
	})

	Convey("Should fail when parsing response from Google fails", t, func() {

		mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(200)
			fmt.Fprintln(w, `{invalidjsonresponse`)
		}))
		defer mockServer.Close()
		setDefaultServer(mockServer)

		requestWrapper := makeValidGoogleRequest()
		_, _, err := HandleSignUp(requestWrapper, &adapters.MongoAdapter{})

		So(err.Code, ShouldEqual, http.StatusInternalServerError)
	})

	Convey("Should fail when Google's response is not as expected", t, func() {

		mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(200)
			// the field 'aud' is removed from the response body
			fmt.Fprintln(w, `{"iss": "accounts.google.com","at_hash": "pSLbt169EwtjApOMQMoTnA","sub": "107419746647224140307","email_verified": "true","azp": "407408718192.apps.googleusercontent.com","hd": "miwi.com","email": "emir@miwi.com","iat": "1449240861","exp": "1449244461","alg": "RS256","kid": "ce30d9f163852843c9a94ce1c1d711e4464d4391"}`)
		}))
		defer mockServer.Close()
		setDefaultServer(mockServer)

		requestWrapper := makeValidGoogleRequest()

		_, _, err := HandleSignUp(requestWrapper, &adapters.MongoAdapter{})

		So(err.Code, ShouldEqual, http.StatusInternalServerError)
	})

	Convey("Should fail when client ids don't match.", t, func() {

		mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(200)
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprintln(w, `{"iss": "accounts.google.com","at_hash": "pSLbt169EwtjApOMQMoTnA","aud": "407408718192.apps.googleusercontent.com","sub": "107419746647224140307","email_verified": "true","azp": "407408718192.apps.googleusercontent.com","hd": "miwi.com","email": "emir@miwi.com","iat": "1449240861","exp": "1449244461","alg": "RS256","kid": "ce30d9f163852843c9a94ce1c1d711e4464d4391"}`)
		}))
		defer mockServer.Close()
		setDefaultServer(mockServer)

		requestWrapper := makeValidGoogleRequest()

		_, _, err := HandleSignUp(requestWrapper, &adapters.MongoAdapter{})

		So(err.Code, ShouldEqual, http.StatusInternalServerError)
	})

	Convey("Should create new account.", t, func() {

		mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(200)
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprintln(w, `{"iss": "accounts.google.com","at_hash": "pSLbt169EwtjApOMQMoTnA","aud": "googleclientid","sub": "107419746647224140307","email_verified": "true","azp": "407408718192.apps.googleusercontent.com","hd": "miwi.com","email": "emir@miwi.com","iat": "1449240861","exp": "1449244461","alg": "RS256","kid": "ce30d9f163852843c9a94ce1c1d711e4464d4391"}`)
		}))
		defer mockServer.Close()
		setDefaultServer(mockServer)

		var called bool
		getAccountData = func(requestWrapper messages.RequestWrapper, dbAdapter *adapters.MongoAdapter) (accountData map[string]interface{}, err *utils.Error) {
			called = true
			// returning nil account info
			return
		}

		requestWrapper := makeValidGoogleRequest()
		response, _, err := HandleSignUp(requestWrapper, &adapters.MongoAdapter{})

		So(err, ShouldBeNil)
		So(called, ShouldBeTrue)
		So(response.Status, ShouldEqual, http.StatusCreated)
	})

	Convey("Should return existing account.", t, func() {

		mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(200)
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprintln(w, `{"iss": "accounts.google.com","at_hash": "pSLbt169EwtjApOMQMoTnA","aud": "googleclientid","sub": "107419746647224140307","email_verified": "true","azp": "407408718192.apps.googleusercontent.com","hd": "miwi.com","email": "emir@miwi.com","iat": "1449240861","exp": "1449244461","alg": "RS256","kid": "ce30d9f163852843c9a94ce1c1d711e4464d4391"}`)
		}))
		defer mockServer.Close()
		setDefaultServer(mockServer)

		var called bool
		getAccountData = func(requestWrapper messages.RequestWrapper, dbAdapter *adapters.MongoAdapter) (accountData map[string]interface{}, err *utils.Error) {
			called = true
			accountData = make(map[string]interface{})
			accountData["_id"] = "564f1a28e63bce219e1cc745"
			// returning existing account
			return
		}

		requestWrapper := makeValidGoogleRequest()
		response, _, err := HandleSignUp(requestWrapper, &adapters.MongoAdapter{})

		So(err, ShouldBeNil)
		So(called, ShouldBeTrue)
		So(response.Status, ShouldEqual, http.StatusCreated)
	})

}

func TestHandleLogin(t *testing.T) {

	Convey("Should return bad request", t, func() {

		var called bool
		getAccountData = func(requestWrapper messages.RequestWrapper, dbAdapter *adapters.MongoAdapter) (accountData map[string]interface{}, err *utils.Error) {
			called = true
			return
		}

		var message messages.Message
		message.Body = make(map[string]interface{})

		var requestWrapper messages.RequestWrapper
		requestWrapper.Message = message

		_, err := HandleLogin(requestWrapper, &adapters.MongoAdapter{})

		So(err.Code, ShouldEqual, http.StatusBadRequest)
		So(called, ShouldBeFalse)
	})

	Convey("Should login with email", t, func() {

		getAccountData = func(requestWrapper messages.RequestWrapper, dbAdapter *adapters.MongoAdapter) (accountData map[string]interface{}, err *utils.Error) {
			accountData = make(map[string]interface{})
			// hased of 'zuhaha'
			accountData["password"] = "$2a$10$wqvcYHiRvoCy5ZUurNz9wuokDH1DyXjfd8k6Hk4DSJKui76gx1yrO"
			accountData["_id"] = "564f1a28e63bce219e1cc745"
			return
		}

		var message messages.Message
		message.Body = make(map[string]interface{})
		message.Body["email"] = "email@domain.com"
		message.Body["password"] = "zuhaha"

		var requestWrapper messages.RequestWrapper
		requestWrapper.Message = message

		response, err := HandleLogin(requestWrapper, &adapters.MongoAdapter{})

		So(err, ShouldBeNil)
		So(response.Status, ShouldEqual, http.StatusOK)
	})

	Convey("Should login with username", t, func() {

		getAccountData = func(requestWrapper messages.RequestWrapper, dbAdapter *adapters.MongoAdapter) (accountData map[string]interface{}, err *utils.Error) {
			accountData = make(map[string]interface{})
			// hased of 'zuhaha'
			accountData["password"] = "$2a$10$wqvcYHiRvoCy5ZUurNz9wuokDH1DyXjfd8k6Hk4DSJKui76gx1yrO"
			accountData["_id"] = "564f1a28e63bce219e1cc745"
			return
		}

		var message messages.Message
		message.Body = make(map[string]interface{})
		message.Body["username"] = "someusername"
		message.Body["password"] = "zuhaha"

		var requestWrapper messages.RequestWrapper
		requestWrapper.Message = message

		response, _ := HandleLogin(requestWrapper, &adapters.MongoAdapter{})
		So(response.Status, ShouldEqual, http.StatusOK)
	})

	Convey("Should return password error", t, func() {

		getAccountData = func(requestWrapper messages.RequestWrapper, dbAdapter *adapters.MongoAdapter) (accountData map[string]interface{}, err *utils.Error) {
			accountData = make(map[string]interface{})
			// hased of 'zuhaha'
			accountData["password"] = "$2a$10$wqvcYHiRvoCy5ZUurNz9wuokDH1DyXjfd8k6Hk4DSJKui76gx1yrO"
			accountData["_id"] = "564f1a28e63bce219e1cc745"
			return
		}

		var message messages.Message
		message.Body = make(map[string]interface{})
		message.Body["email"] = "email@domain.com"
		message.Body["password"] = "notzuhaha"

		var requestWrapper messages.RequestWrapper
		requestWrapper.Message = message

		response, _ := HandleLogin(requestWrapper, &adapters.MongoAdapter{})
		So(response.Status, ShouldEqual, http.StatusUnauthorized)
	})

	Convey("Should return error if account doesn't exist", t, func() {

		getAccountData = func(requestWrapper messages.RequestWrapper, dbAdapter *adapters.MongoAdapter) (accountData map[string]interface{}, err *utils.Error) {
			err = &utils.Error{http.StatusNotFound, "Item not found."}
			return
		}

		var message messages.Message
		message.Body = make(map[string]interface{})
		message.Body["email"] = "email@domain.com"
		message.Body["password"] = "apasswordimpossibletofind"

		var requestWrapper messages.RequestWrapper
		requestWrapper.Message = message

		_, err := HandleLogin(requestWrapper, &adapters.MongoAdapter{})
		So(err.Code, ShouldEqual, http.StatusUnauthorized)
	})

	Convey("Should return token generation error", t, func() {

		getAccountData = func(requestWrapper messages.RequestWrapper, dbAdapter *adapters.MongoAdapter) (accountData map[string]interface{}, err *utils.Error) {
			accountData = make(map[string]interface{})
			// hased of 'zuhaha'
			accountData["password"] = "$2a$10$wqvcYHiRvoCy5ZUurNz9wuokDH1DyXjfd8k6Hk4DSJKui76gx1yrO"
			accountData["_id"] = "564f1a28e63bce219e1cc745"
			return
		}

		generateToken = func(userId string, userData map[string]interface{}) (tokenString string, err *utils.Error) {
			err = &utils.Error{http.StatusInternalServerError, "Generating token failed."}
			return
		}

		var message messages.Message
		message.Body = make(map[string]interface{})
		message.Body["email"] = "email@domain.com"
		message.Body["password"] = "zuhaha"

		var requestWrapper messages.RequestWrapper
		requestWrapper.Message = message

		_, err := HandleLogin(requestWrapper, &adapters.MongoAdapter{})
		So(err.Code, ShouldEqual, http.StatusInternalServerError)
	})

}

func TestResetPassword(t *testing.T) {

	config.SystemConfig = config.Config{}
	config.SystemConfig.ResetPassword = map[string]string {
		"senderEmail": "info@rihtim.com",
		"senderEmailPassword": "emailadresspassword",
		"smtpServer":"mail.rihtim.com",
		"smtpPort":"25",
		"mailSubject":"Reset password!",
		"mailContentTemplate":"Your new password is %s.",
	}

	Convey("Should return internal server error", t, func() {

		correctConfig := config.SystemConfig.ResetPassword
		config.SystemConfig.ResetPassword = nil

		var message messages.Message
		message.Body = make(map[string]interface{})

		var requestWrapper messages.RequestWrapper
		requestWrapper.Message = message

		_, err := HandleResetPassword(requestWrapper, &adapters.MongoAdapter{})

		So(err.Code, ShouldEqual, http.StatusInternalServerError)

		config.SystemConfig.ResetPassword = correctConfig
	})

	Convey("Should return bad request", t, func() {

		var message messages.Message
		message.Body = make(map[string]interface{})

		var requestWrapper messages.RequestWrapper
		requestWrapper.Message = message

		_, err := HandleResetPassword(requestWrapper, &adapters.MongoAdapter{})

		So(err.Code, ShouldEqual, http.StatusBadRequest)
	})

	Convey("Should return not found", t, func() {

		getAccountData = func(requestWrapper messages.RequestWrapper, dbAdapter *adapters.MongoAdapter) (accountData map[string]interface{}, err *utils.Error) {
			err = &utils.Error{http.StatusNotFound, "Account not found."}
			return
		}

		var message messages.Message
		message.Body = make(map[string]interface{})
		message.Body["email"] = "email@domain.com"

		var requestWrapper messages.RequestWrapper
		requestWrapper.Message = message

		_, err := HandleResetPassword(requestWrapper, &adapters.MongoAdapter{})

		So(err.Code, ShouldEqual, http.StatusNotFound)
	})

	Convey("Should return the error from adapters.HandlePut", t, func() {

		getAccountData = func(requestWrapper messages.RequestWrapper, dbAdapter *adapters.MongoAdapter) (accountData map[string]interface{}, err *utils.Error) {
			accountData = make(map[string]interface{})
			accountData["_id"] = "564f1a28e63bce219e1cc745"
			return
		}

		adapters.Update = func(collection string, id string, data map[string]interface{}) (response map[string]interface{}, hookBody map[string]interface{}, err *utils.Error) {
			err = &utils.Error{http.StatusInternalServerError, "Some error happened."}
			return
		}

		var message messages.Message
		message.Body = make(map[string]interface{})
		message.Body["email"] = "email@domain.com"

		var requestWrapper messages.RequestWrapper
		requestWrapper.Message = message

		_, err := HandleResetPassword(requestWrapper, &adapters.MongoAdapter{})

		So(err, ShouldNotBeNil)
	})

	Convey("Should call adapters.HandlePut with new password and correct user resource", t, func() {

		getAccountData = func(requestWrapper messages.RequestWrapper, dbAdapter *adapters.MongoAdapter) (accountData map[string]interface{}, err *utils.Error) {
			accountData = make(map[string]interface{})
			accountData["_id"] = "564f1a28e63bce219e1cc745"
			return
		}

		var isResCorrect bool
		var isPasswordProvided bool
		adapters.Update = func(collection string, id string, data map[string]interface{}) (response map[string]interface{}, hookBody map[string]interface{}, err *utils.Error) {
			isResCorrect = strings.EqualFold("users", collection) && strings.EqualFold("564f1a28e63bce219e1cc745", id)
			isPasswordProvided = len(data["password"].(string)) > 0
			return
		}

		var called bool
		sendNewPasswordEmail = func(smtpServer, smtpPost, senderEmail, senderEmailPassword, subject, contentTemplate, recipientEmail, newPassword string) (err *utils.Error) {
			called = true
			return
		}

		var message messages.Message
		message.Body = make(map[string]interface{})
		message.Body["email"] = "email@domain.com"

		var requestWrapper messages.RequestWrapper
		requestWrapper.Message = message

		HandleResetPassword(requestWrapper, &adapters.MongoAdapter{})

		So(isResCorrect, ShouldBeTrue)
		So(isPasswordProvided, ShouldBeTrue)
		So(called, ShouldBeTrue)
	})

}

