package corefunctions

import (
	"fmt"
	"strings"
	"net/smtp"
	"net/http"
	"math/rand"
	"encoding/json"
	"golang.org/x/crypto/bcrypt"
	"github.com/rihtim/core/utils"
	"github.com/rihtim/core/auth"
	"github.com/rihtim/core/messages"
	"github.com/rihtim/core/database"
	"github.com/rihtim/core/constants"
)

/**
 * Reset password configuration: should contain these fields:
    "senderEmail": Sender email address.
    "senderEmailPassword": Sender email address's password.
    "smtpServer": Smtp server to call.
    "smtpPort": Smtp port to call.
    "mailSubject": Subject text of the mail.
    "mailContentTemplate": HTML template for the email content. Provide %s in the place where the password will be shown.
     	For Ex: "<html><head></head><body><p>Dear Rihtim user,</p> <p>As you requested, a new password is generated for you. You can use the password below to login. </p><p><b>%s</b> </p><p>Please change your password with something you choose after your first login with this generated password. </p><p>Thanks,<br/>Rihtim Team</p></body></html>"
 */
var ResetPasswordConfig    map[string]string

// used for password generation
var fruits = []string{"apples", "appricots", "avocados", "bananas", "cherries", "coconuts", "damsons",
	"dates", "durian", "grapes", "guavas", "jambuls", "jujubes", "kiwis", "lemons", "limes", "mangos", "melons",
	"olives", "oranges", "papayas", "peaches", "pears", "plums", "pumpkins", "pomelos", "satsumas", "tomatoes"}
var quantities = []string{"two", "three", "four", "five", "six", "seven", "eight", "nine", "ten"}

var Register = func(user interface{}, message messages.Message) (response messages.Message, hookBody map[string]interface{}, err *utils.Error) {

	_, hasEmail := message.Body["email"]
	password, hasPassword := message.Body["password"]

	if !hasEmail || !hasPassword {
		err = &utils.Error{http.StatusBadRequest, "Email and password must be provided."}
		return
	}

	existingAccount, _ := getAccountData(message)
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

	response.Body, hookBody, err = database.Adapter.Create(constants.ClassUsers, message.Body)
	if err != nil {
		return
	}

	accessToken, tokenErr := auth.GenerateToken(response.Body[constants.IdIdentifier].(string), response.Body)
	if tokenErr != nil {
		err = tokenErr
		return
	}

	delete(response.Body, "password")
	response.Status = http.StatusCreated
	response.Body["accessToken"] = accessToken
	return
}

var Login = func(user interface{}, message messages.Message) (response messages.Message, hookBody map[string]interface{}, err *utils.Error) {

	_, hasEmail := message.Body["email"]
	password, hasPassword := message.Body["password"]

	if !hasEmail || !hasPassword {
		err = &utils.Error{http.StatusBadRequest, "Login request must contain email and password."}
		return
	}

	accountData, getAccountErr := getAccountData(message)
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

		var accessToken string
		accessToken, err = auth.GenerateToken(accountData[constants.IdIdentifier].(string), accountData)
		if err == nil {
			response.Body["accessToken"] = accessToken
			response.Status = http.StatusOK
		}
	} else {
		response.Status = http.StatusUnauthorized
	}
	return
}

var ChangePassword = func(user interface{}, message messages.Message) (response messages.Message, hookBody map[string]interface{}, err *utils.Error) {

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

var ResetPassword = func(user interface{}, message messages.Message) (response messages.Message, hookBody map[string]interface{}, err *utils.Error) {

	if ResetPasswordConfig == nil {
		err = &utils.Error{http.StatusInternalServerError, "Email reset configuration is not defined."}
	}

	senderEmail, hasSenderEmail := ResetPasswordConfig["senderEmail"]
	senderEmailPassword, hasSenderEmailPassword := ResetPasswordConfig["senderEmailPassword"]
	smtpServer, hasSmtpServer := ResetPasswordConfig["smtpServer"]
	smtpPort, hasSmtpPort := ResetPasswordConfig["smtpPort"]
	mailSubject, hasMailSubject := ResetPasswordConfig["mailSubject"]
	mailContentTemplate, hasMailContent := ResetPasswordConfig["mailContentTemplate"]

	if !hasSmtpServer || !hasSmtpPort || !hasSenderEmail || !hasSenderEmailPassword || !hasMailSubject || !hasMailContent {
		err = &utils.Error{http.StatusInternalServerError, "Email reset configuration is not correct."}
		return
	}

	recipientEmail, hasRecipientEmail := message.Body["email"]
	if !hasRecipientEmail {
		err = &utils.Error{http.StatusBadRequest, "Email must be provided in the body."}
		return
	}

	accountData, err := getAccountData(message)
	if err != nil {
		return
	}

	// generating random password like: "twoapplesandfiveoranges" or "threekiwisandsevenbananas"
	passwordFirstHalf := quantities[rand.Intn(len(quantities))] + fruits[rand.Intn(len(fruits))]
	passwordSecondHalf := quantities[rand.Intn(len(quantities))] + fruits[rand.Intn(len(fruits))]
	generatedPassword := passwordFirstHalf + "and" + passwordSecondHalf
	hashedPassword, hashErr := bcrypt.GenerateFromPassword([]byte(generatedPassword), bcrypt.DefaultCost)
	if hashErr != nil {
		err = &utils.Error{http.StatusInternalServerError, "Hashing new password failed. Reason: " + hashErr.Error()}
		return
	}

	body := map[string]interface{}{"password": string(hashedPassword)}
	response.Body, _, err = database.Adapter.Update(constants.ClassUsers, accountData[constants.IdIdentifier].(string), body)
	if err != nil {
		return
	}

	err = sendNewPasswordEmail(smtpServer, smtpPort, senderEmail, senderEmailPassword, mailSubject, mailContentTemplate, recipientEmail.(string), generatedPassword)
	return
}

var GrantRole = func(user interface{}, message messages.Message) (response messages.Message, hookBody map[string]interface{}, err *utils.Error) {

	if len(user.(map[string]interface{})) == 0 {
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
			if !arrayContains(roles.([]interface{}), roleToGrant) {
				roles = append(roles.([]interface{}), roleToGrant)
			}
		}
	}

	body := map[string]interface{}{constants.RolesIdentifier: roles}
	response.Body, hookBody, err = database.Adapter.Update(constants.ClassUsers, userIdToUpdate, body)
	response.Body[constants.RolesIdentifier] = roles
	return
}

var RecallRole = func(user interface{}, message messages.Message) (response messages.Message, hookBody map[string]interface{}, err *utils.Error) {

	if len(user.(map[string]interface{})) == 0 {
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

			if !arrayContains(rolesToRecall.([]interface{}), existingRole) {
				newRoles = append(newRoles, existingRole)
			}
		}
	}

	body := map[string]interface{}{constants.RolesIdentifier: newRoles}
	response.Body, hookBody, err = database.Adapter.Update(constants.ClassUsers, userIdToUpdate, body)
	response.Body[constants.RolesIdentifier] = newRoles
	return
}

var getAccountData = func(message messages.Message) (accountData map[string]interface{}, err *utils.Error) {

	var whereParams = make(map[string]interface{})
	var queryKey, queryParam string

	if username, hasUsername := message.Body["username"]; hasUsername && username != "" {
		queryKey = "username"
		queryParam = username.(string)
	} else if email, hasEmail := message.Body["email"]; hasEmail && email != "" {
		queryKey = "email"
		queryParam = email.(string)
	} else if facebookData, hasFacebookData := message.Body["facebook"]; hasFacebookData {
		facebookDataAsMap := facebookData.(map[string]interface{})
		queryParam = facebookDataAsMap["id"].(string)
		queryKey = "facebook.id"
	} else if googleData, hasGoogleData := message.Body["google"]; hasGoogleData {
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
	message.Parameters["where"] = []string{string(whereParamsJson)}

	results, fetchErr := database.Adapter.Query(constants.ClassUsers, message.Parameters)
	resultsAsMap := results[constants.ListIdentifier].([]map[string]interface{})
	if fetchErr != nil || len(resultsAsMap) == 0 {
		err = &utils.Error{http.StatusNotFound, "Account not found."}
		return
	}
	accountData = resultsAsMap[0]

	return
}

var sendNewPasswordEmail = func(smtpServer, smtpPost, senderEmail, senderEmailPassword, subject, contentTemplate, recipientEmail, newPassword string) (err *utils.Error) {

	auth := smtp.PlainAuth("", senderEmail, senderEmailPassword, smtpServer)

	generatedContent := fmt.Sprintf(contentTemplate, newPassword)
	to := []string{recipientEmail}
	msg := []byte(
	"From: " + senderEmail + "\r\n" +
	"To: " + recipientEmail + "\r\n" +
	"Subject: " + subject + "\r\n" +
	"MIME-version: 1.0;\nContent-Type: text/html; charset=\"UTF-8\";\n\n" +
	"\r\n" + generatedContent + "\r\n")
	sendMailErr := smtp.SendMail(smtpServer + ":" + smtpPost, auth, senderEmail, to, msg)

	if sendMailErr != nil {
		err = &utils.Error{http.StatusInternalServerError, "Sending email failed. Reason: " + sendMailErr.Error()}
	}
	return
}

var arrayContains = func(array []interface{}, item interface{}) (contains bool) {
	set := make(map[string]bool)
	for _, v := range array {
		set[v.(string)] = true
	}
	_, contains = set[item.(string)]
	return
}