package corefunctions

import (
	"fmt"
	"strings"
	"net/http"
	"math/rand"
	"encoding/json"
	"golang.org/x/crypto/bcrypt"
	"github.com/rihtim/core/utils"
	"github.com/rihtim/core/auth"
	"github.com/rihtim/core/messages"
	"github.com/rihtim/core/database"
	"github.com/rihtim/core/constants"
	"github.com/rihtim/core/validator"
	"gopkg.in/mgo.v2/bson"
	"github.com/rihtim/core/keys"
	"reflect"
	"time"
	"github.com/go-gomail/gomail"
)

var ResetPasswordConfig    map[string]string

var fieldsForRegister = map[string]bool{
	constants.IdIdentifier: false,
	constants.AclIdentifier: false,
	"createdAt": false,
	"updatedAt": false,
	"email": true,
	"password": true,
}

const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890"
const (
	letterIdxBits = 6                    // 6 bits to represent a letter index
	letterIdxMask = 1 << letterIdxBits - 1 // All 1-bits, as many as letterIdxBits
	letterIdxMax = 63 / letterIdxBits   // # of letter indices fitting in 63 bits
)

// used for password generation
var src = rand.NewSource(time.Now().UnixNano())

var GenerateRandomString = func(n int) string {
	b := make([]byte, n)
	// A src.Int63() generates 63 random bits, enough for letterIdxMax characters!
	for i, cache, remain := n - 1, src.Int63(), letterIdxMax; i >= 0; {
		if remain == 0 {
			cache, remain = src.Int63(), letterIdxMax
		}
		if idx := int(cache & letterIdxMask); idx < len(letterBytes) {
			b[i] = letterBytes[idx]
			i--
		}
		cache >>= letterIdxBits
		remain--
	}

	return string(b)
}

var Register = func(user interface{}, message messages.Message) (response messages.Message, finalInterceptorBody map[string]interface{}, err *utils.Error) {

	err = validator.ValidateInputFields(fieldsForRegister, message.Body)
	if err != nil {
		return
	}
	password := message.Body["password"]

	existingAccount, _ := getAccountData(message.Body)
	if existingAccount != nil {
		err = &utils.Error{http.StatusConflict, "User with same email already exists."}
		return
	}

	hashedPassword, hashErr := bcrypt.GenerateFromPassword([]byte(password.(string)), bcrypt.DefaultCost)
	if hashErr != nil {
		err = &utils.Error{http.StatusInternalServerError, "Hashing password failed."}
		return
	}
	message.Body["password"] = string(hashedPassword)

	id := bson.NewObjectId().Hex()
	userRole := "user:" + id
	message.Body["_id"] = id
	message.Body[constants.AclIdentifier] = map[string]interface{}{
		constants.All: map[string]bool{
			"get": true,
		},
		userRole: map[string]bool{
			"get": true,
			"update": true,
		},
	}

	response.Body, finalInterceptorBody, err = database.Adapter.Create(constants.ClassUsers, message.Body)
	if err != nil {
		return
	}

	userData := map[string]interface{}{
		"userId": response.Body[constants.IdIdentifier].(string),
	}
	authData, adErr := auth.Adapter.GenerateAuthData(userData)
	if adErr != nil {
		err = adErr
		return
	}

	accessToken, hasToken := authData["token"].(string)
	if !hasToken {
		err = &utils.Error{http.StatusInternalServerError, "Token couldn't be retrieved from auth data."}
		return
	}

	delete(response.Body, "password")
	response.Status = http.StatusCreated
	response.Body["accessToken"] = accessToken
	return
}

var Login = func(user interface{}, message messages.Message) (response messages.Message, finalInterceptorBody map[string]interface{}, err *utils.Error) {

	_, hasEmail := message.Body["email"]
	password, hasPassword := message.Body["password"]

	if !hasEmail || !hasPassword {
		err = &utils.Error{http.StatusBadRequest, "Login request must contain email and password."}
		return
	}

	accountData, getAccountErr := getAccountData(message.Body)
	if getAccountErr != nil {
		err = getAccountErr
		if getAccountErr.Code == http.StatusNotFound {
			err = &utils.Error{http.StatusUnauthorized, "Credentials don't match or account doesn't exist."}
		}
		return
	}
	existingPassword := accountData["password"].(string)

	passwordError := bcrypt.CompareHashAndPassword([]byte(existingPassword), []byte(password.(string)))
	if passwordError == nil {
		delete(accountData, "password")
		response.Body = accountData

		userData := map[string]interface{}{
			"userId": accountData[constants.IdIdentifier].(string),
		}
		authData, adErr := auth.Adapter.GenerateAuthData(userData)
		if adErr != nil {
			err = adErr
			return
		}

		accessToken, hasToken := authData["token"].(string)
		if !hasToken {
			err = &utils.Error{http.StatusInternalServerError, "Token couldn't be retrieved from auth data."}
			return
		}

		response.Body["accessToken"] = accessToken
		response.Status = http.StatusOK
	} else {
		response.Status = http.StatusUnauthorized
	}
	return
}

var ChangePassword = func(user interface{}, message messages.Message) (response messages.Message, finalInterceptorBody map[string]interface{}, err *utils.Error) {

	userAsMap := user.(map[string]interface{})

	if len(userAsMap) == 0 {
		err = &utils.Error{http.StatusUnauthorized, "Access token must be provided for change password request."}
		return
	}

	password, hasPassword := message.Body["password"]
	if !hasPassword {
		err = &utils.Error{http.StatusBadRequest, "Password must be provided in the body with field 'password'."}
		return
	}

	newPassword, hasNewPassword := message.Body["newPassword"]
	if !hasNewPassword {
		err = &utils.Error{http.StatusBadRequest, "New password must be provided in the body with field 'newPassword'."}
		return
	}

	existingPassword := userAsMap["password"].(string)

	passwordError := bcrypt.CompareHashAndPassword([]byte(existingPassword), []byte(password.(string)))
	if passwordError != nil {
		err = &utils.Error{http.StatusUnauthorized, "Existing password is not correct."}
		return
	}

	hashedPassword, hashErr := bcrypt.GenerateFromPassword([]byte(newPassword.(string)), bcrypt.DefaultCost)
	if hashErr != nil {
		err = &utils.Error{http.StatusInternalServerError, "Hashing new password failed. Reason: " + hashErr.Error()}
		return
	}

	body := map[string]interface{}{"password": string(hashedPassword)}
	response.Body, _, err = database.Adapter.Update(constants.ClassUsers, userAsMap[constants.IdIdentifier].(string), body)
	return
}

/**
 * Used to reset user's password.
 */
var ResetPassword = func(userInfo map[string]interface{}) (password string, err *utils.Error) {

	accountData, err := getAccountData(userInfo)
	if err != nil {
		return
	}

	// generating random password
	generatedPassword := GenerateRandomString(6)
	hashedPassword, hashErr := bcrypt.GenerateFromPassword([]byte(generatedPassword), bcrypt.DefaultCost)
	if hashErr != nil {
		err = &utils.Error{http.StatusInternalServerError, "Hashing new password failed. Reason: " + hashErr.Error()}
		return
	}

	body := map[string]interface{}{"password": string(hashedPassword)}
	_, _, err = database.Adapter.Update(constants.ClassUsers, accountData[constants.IdIdentifier].(string), body)
	if err != nil {
		return
	}

	password = generatedPassword
	return
}

var GrantRole = func(user interface{}, message messages.Message) (response messages.Message, finalInterceptorBody map[string]interface{}, err *utils.Error) {

	// check whether the headers give special permissions to perform the request
	var isGrantedByKey bool
	isGrantedByKey, err = keys.Adapter.CheckKeyPermissions(message.Headers)
	if err != nil {
		return
	}

	if !isGrantedByKey && len(user.(map[string]interface{})) == 0 {
		err = &utils.Error{http.StatusUnauthorized, "Grant role request requires access token."}
		return
	}

	resParts := strings.Split(message.Res, "/")
	if len(resParts) != 4 || !strings.EqualFold(resParts[1], constants.ClassUsers) {
		err = &utils.Error{http.StatusBadRequest, "Grant role can only be used on user objects. Ex: '/users/{id}/grantRole'"}
		return
	}
	userIdToUpdate := resParts[2]

	if message.Body == nil {
		err = &utils.Error{http.StatusBadRequest, "Grant role request must contain body."}
		return
	}

	rolesToGrant, hasRolesToGrant := message.Body[constants.RolesIdentifier]
	if !hasRolesToGrant {
		err = &utils.Error{http.StatusBadRequest, "Grant role request must contain list of roles in '_roles' field in body."}
		return
	}

	if !isGrantedByKey {
		userAsMap := user.(map[string]interface{})
		requestOwnersRoles, requestOwnersHasRoles := userAsMap[constants.RolesIdentifier]
		if !requestOwnersHasRoles {
			err = &utils.Error{http.StatusUnauthorized, "Request owner doesn't have any role info."}
			return
		}

		matchingRoleCount := 0
		for _, roleToGrant := range rolesToGrant.([]interface{}) {
			for _, userRole := range requestOwnersRoles.([]interface{}) {
				if strings.EqualFold(roleToGrant.(string), userRole.(string)) {
					matchingRoleCount++
					continue
				}
			}
		}

		if matchingRoleCount != len(rolesToGrant.([]interface{})) {
			err = &utils.Error{http.StatusUnauthorized, "Request owner doesn't have enough permissions to grant the given roles."}
			return
		}
	}

	var userToUpdate map[string]interface{}
	userToUpdate, err = database.Adapter.Get(constants.ClassUsers, userIdToUpdate)
	if err != nil {
		return
	}

	roles, hasRoles := userToUpdate[constants.RolesIdentifier]

	if !hasRoles {
		roles = rolesToGrant
	} else {

		for _, roleToGrant := range rolesToGrant.([]interface{}) {
			if !arrayContainsString(roles.([]interface{}), roleToGrant) {
				roles = append(roles.([]interface{}), roleToGrant)
			}
		}
	}

	body := map[string]interface{}{constants.RolesIdentifier: roles}
	response.Body, finalInterceptorBody, err = database.Adapter.Update(constants.ClassUsers, userIdToUpdate, body)
	response.Body[constants.RolesIdentifier] = roles
	return
}

var RecallRole = func(user interface{}, message messages.Message) (response messages.Message, finalInterceptorBody map[string]interface{}, err *utils.Error) {

	// check whether the headers give special permissions to perform the request
	var isGrantedByKey bool
	isGrantedByKey, err = keys.Adapter.CheckKeyPermissions(message.Headers)
	if err != nil {
		return
	}

	if !isGrantedByKey && len(user.(map[string]interface{})) == 0 {
		err = &utils.Error{http.StatusUnauthorized, "Grant role request requires access token."}
		return
	}

	resParts := strings.Split(message.Res, "/")
	if len(resParts) != 4 || !strings.EqualFold(resParts[1], constants.ClassUsers) {
		err = &utils.Error{http.StatusBadRequest, "Recall role can only be used on user objects. Ex: '/users/{id}/recallRole'"}
		return
	}
	userIdToUpdate := resParts[2]

	if message.Body == nil {
		err = &utils.Error{http.StatusBadRequest, "Recall role request must contain body."}
		return
	}

	rolesToRecall, hasRolesToRecall := message.Body[constants.RolesIdentifier]
	if !hasRolesToRecall {
		err = &utils.Error{http.StatusBadRequest, "Recall role request must contain list of roles in '_roles' field in body."}
		return
	}

	if !isGrantedByKey {
		userAsMap := user.(map[string]interface{})
		requestOwnersRoles, requestOwnersHasRoles := userAsMap[constants.RolesIdentifier]
		if !requestOwnersHasRoles {
			err = &utils.Error{http.StatusUnauthorized, "Request owner doesn't have any role info."}
			return
		}

		matchingRoleCount := 0
		for _, roleToRecall := range rolesToRecall.([]interface{}) {
			for _, userRole := range requestOwnersRoles.([]interface{}) {
				if strings.EqualFold(roleToRecall.(string), userRole.(string)) {
					matchingRoleCount++
					continue
				}
			}
		}

		if matchingRoleCount != len(rolesToRecall.([]interface{})) {
			err = &utils.Error{http.StatusUnauthorized, "Request owner doesn't have enough permissions to recall the given roles."}
			return
		}
	}

	var userToUpdate map[string]interface{}
	userToUpdate, err = database.Adapter.Get(constants.ClassUsers, userIdToUpdate)
	if err != nil {
		return
	}

	existingRoles, hasRoles := userToUpdate[constants.RolesIdentifier]
	newRoles := make([]interface{}, 0)

	if !hasRoles {
		response.Body = map[string]interface{}{"message": "User doesn't have any role info. Not updating anything."}
		return
	} else {

		for _, existingRole := range existingRoles.([]interface{}) {

			if !arrayContainsString(rolesToRecall.([]interface{}), existingRole) {
				newRoles = append(newRoles, existingRole)
			}
		}
	}

	body := map[string]interface{}{constants.RolesIdentifier: newRoles}
	response.Body, finalInterceptorBody, err = database.Adapter.Update(constants.ClassUsers, userIdToUpdate, body)
	response.Body[constants.RolesIdentifier] = newRoles
	return
}

var Append = func(user interface{}, message messages.Message) (response messages.Message, finalInterceptorBody map[string]interface{}, err *utils.Error) {

	// check whether the headers give special permissions to perform the request
	var isGrantedByKey bool
	isGrantedByKey, err = keys.Adapter.CheckKeyPermissions(message.Headers)
	if err != nil {
		return
	}

	fmt.Println(message.Headers)
	if !isGrantedByKey && len(user.(map[string]interface{})) == 0 {
		err = &utils.Error{http.StatusUnauthorized, "Append request requires access token."}
		return
	}

	resParts := strings.Split(message.Res, "/")
	if len(resParts) != 5 {
		err = &utils.Error{http.StatusBadRequest, "Append can only be used on array fields. Ex: '/groups/{id}/members/appendUnique'"}
		return
	}
	itemClassToUpdate := resParts[1]
	itemIdToUpdate := resParts[2]
	fieldToAppend := resParts[3]

	if message.Body == nil {
		err = &utils.Error{http.StatusBadRequest, "Append request must contain body."}
		return
	}

	itemsToAdd, hasItemsToAdd := message.Body["items"]
	if !hasItemsToAdd {
		err = &utils.Error{http.StatusBadRequest, "Append request must contain list of items in 'items' field in body."}
		return
	}

	var itemToUpdate map[string]interface{}
	itemToUpdate, err = database.Adapter.Get(itemClassToUpdate, itemIdToUpdate)
	if err != nil {
		return
	}

	fieldValueToAppend, hasFieldToAppend := itemToUpdate[fieldToAppend]

	if !hasFieldToAppend {
		fieldValueToAppend = make([]interface{}, 0)
	} else if fieldObjectType := reflect.TypeOf(fieldValueToAppend); fieldObjectType.Kind() != reflect.Slice {
		err = &utils.Error{http.StatusBadRequest, "The field '" + fieldToAppend + "' is not an array."}
		return
	}

	for _, itemToAdd := range itemsToAdd.([]interface{}) {
		if itemType := reflect.TypeOf(itemToAdd); itemType.Kind() == reflect.Map {
			if i := arrayContainsMap(fieldValueToAppend.([]interface{}), itemToAdd.(map[string]interface{})); i == -1 {
				fieldValueToAppend = append(fieldValueToAppend.([]interface{}), itemToAdd)
			}
		}
		if itemType := reflect.TypeOf(itemToAdd); itemType.Kind() == reflect.String {
			if !arrayContainsString(fieldValueToAppend.([]interface{}), itemToAdd) {
				fieldValueToAppend = append(fieldValueToAppend.([]interface{}), itemToAdd)
			}
		}
	}

	body := make(map[string]interface{})
	body[fieldToAppend] = fieldValueToAppend
	response.Body, finalInterceptorBody, err = database.Adapter.Update(itemClassToUpdate, itemIdToUpdate, body)
	response.Body[fieldToAppend] = fieldValueToAppend
	return
}

var Remove = func(user interface{}, message messages.Message) (response messages.Message, finalInterceptorBody map[string]interface{}, err *utils.Error) {

	// check whether the headers give special permissions to perform the request
	var isGrantedByKey bool
	isGrantedByKey, err = keys.Adapter.CheckKeyPermissions(message.Headers)
	if err != nil {
		return
	}

	fmt.Println(message.Headers)
	if !isGrantedByKey && len(user.(map[string]interface{})) == 0 {
		err = &utils.Error{http.StatusUnauthorized, "Remove request requires authentication."}
		return
	}

	resParts := strings.Split(message.Res, "/")
	if len(resParts) != 5 {
		err = &utils.Error{http.StatusBadRequest, "Remove can only be used on array fields. Ex: '/groups/{id}/members/appendUnique'"}
		return
	}
	itemClassToUpdate := resParts[1]
	itemIdToUpdate := resParts[2]
	fieldToRemove := resParts[3]

	if message.Body == nil {
		err = &utils.Error{http.StatusBadRequest, "Remove request must contain body."}
		return
	}

	itemsToRemove, hasItemsToRemove := message.Body["items"]
	if !hasItemsToRemove {
		err = &utils.Error{http.StatusBadRequest, "Remove request must contain list of items in 'items' field in body."}
		return
	}

	var itemToUpdate map[string]interface{}
	itemToUpdate, err = database.Adapter.Get(itemClassToUpdate, itemIdToUpdate)
	if err != nil {
		return
	}

	fieldValueToRemove, hasFieldToAppend := itemToUpdate[fieldToRemove]

	if !hasFieldToAppend {
		err = &utils.Error{http.StatusBadRequest, "The field '" + fieldToRemove + "' doesn't exist."}
		return
	} else if fieldObjectType := reflect.TypeOf(fieldValueToRemove); fieldObjectType.Kind() != reflect.Slice {
		err = &utils.Error{http.StatusBadRequest, "The field '" + fieldToRemove + "' is not an array."}
		return
	}

	var newArray = make([]interface{}, 0)
	for _, existingItem := range fieldValueToRemove.([]interface{}) {
		if itemType := reflect.TypeOf(existingItem); itemType.Kind() == reflect.Map {
			if i := arrayContainsMap(itemsToRemove.([]interface{}), existingItem.(map[string]interface{})); i == -1 {
				newArray = append(newArray, existingItem)
			}
		}
		if itemType := reflect.TypeOf(existingItem); itemType.Kind() == reflect.String {
			if !arrayContainsString(itemsToRemove.([]interface{}), existingItem) {
				newArray = append(newArray, existingItem)
			}
		}
	}

	body := make(map[string]interface{})
	body[fieldToRemove] = newArray
	response.Body, finalInterceptorBody, err = database.Adapter.Update(itemClassToUpdate, itemIdToUpdate, body)
	response.Body[fieldToRemove] = newArray
	return
}

/**
 * Returns account data.
 *
 * param userInfo: information to retrieve the user's full data. must contain one of 'username', 'email',
 * 'facebook' or 'google' info
 */
var getAccountData = func(userInfo map[string]interface{}) (accountData map[string]interface{}, err *utils.Error) {

	var whereParams = make(map[string]interface{})
	var queryKey, queryParam string

	if username, hasUsername := userInfo["username"]; hasUsername && username != "" {
		queryKey = "username"
		queryParam = username.(string)
	} else if email, hasEmail := userInfo["email"]; hasEmail && email != "" {
		queryKey = "email"
		queryParam = email.(string)
	} else if facebookData, hasFacebookData := userInfo["facebook"]; hasFacebookData {
		facebookDataAsMap := facebookData.(map[string]interface{})
		queryParam = facebookDataAsMap["id"].(string)
		queryKey = "facebook.id"
	} else if googleData, hasGoogleData := userInfo["google"]; hasGoogleData {
		googleDataAsMap := googleData.(map[string]interface{})
		queryParam = googleDataAsMap["id"].(string)
		queryKey = "google.id"
	}

	query := make(map[string]string)
	query["$eq"] = queryParam
	whereParams[queryKey] = query

	whereParamsJson, jsonErr := json.Marshal(whereParams)
	if jsonErr != nil {
		err = &utils.Error{http.StatusInternalServerError, "Creating user request failed."}
		return
	}

	parameters := map[string][]string{
		"where": []string{string(whereParamsJson)},
	}

	results, fetchErr := database.Adapter.Query(constants.ClassUsers, parameters)
	resultsAsMap := results[constants.ListIdentifier].([]map[string]interface{})
	if fetchErr != nil || len(resultsAsMap) == 0 {
		err = &utils.Error{http.StatusNotFound, "Account not found."}
		return
	}
	accountData = resultsAsMap[0]

	return
}

var arrayContainsString = func(array []interface{}, item interface{}) (contains bool) {
	set := make(map[string]bool)
	for _, v := range array {
		set[v.(string)] = true
	}
	_, contains = set[item.(string)]
	return
}

var arrayContainsMap = func(array []interface{}, item map[string]interface{}) (index int) {

	for i, itemToCheck := range array {
		if (isMapEquals(itemToCheck.(map[string]interface{}), item)) {
			return i
		}
	}

	return -1
}

var isMapEquals = func(m1, m2 map[string]interface{}) bool {

	// skip if field counts are not equal
	if len(m1) != len(m2) {
		return false
	}

	// check each field of items
	var hasDifferentField = false
	for key, field := range m1 {
		sameFieldValue, hasSameField := m2[key]
		if !hasSameField || sameFieldValue != field {
			hasDifferentField = true
		}
	}

	return !hasDifferentField
}

func sendEmail(smtpServer, senderEmail, senderName, senderEmailPassword string, smtpPort int, to, cc, bcc []string, subject, content string) *utils.Error {

	m := gomail.NewMessage()
	m.SetHeader("From", m.FormatAddress(senderEmail, senderName))
	m.SetHeader("To", to...)
	m.SetHeader("Cc", cc...)
	m.SetHeader("Bcc", bcc...)
	m.SetHeader("Subject", subject)
	m.SetBody("text/html", content)

	d := gomail.NewDialer(smtpServer, smtpPort, senderEmail, senderEmailPassword)

	if err := d.DialAndSend(m); err != nil {
		return &utils.Error{http.StatusInternalServerError, err.Error()}
	}
	return nil
}