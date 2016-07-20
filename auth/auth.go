package auth

import (
	"time"
	"strings"
	"net/http"
	"github.com/rihtim/core/messages"
	"github.com/rihtim/core/utils"
	"github.com/rihtim/core/database"
	"github.com/dgrijalva/jwt-go"
	"github.com/rihtim/core/constants"
)

var commandPermissionMap = map[string]map[string]bool{
	"get": {
		"get": true,
		"query": true,
	},
	"post": {
		"create": true,
	},
	"put": {
		"update": true,
	},
	"delete": {
		"delete": true,
	},
};

var IsGranted = func(collection string, requestWrapper messages.RequestWrapper) (isGranted bool, user map[string]interface{}, err *utils.Error) {

	// grant the request for everyone for file resources
	res := requestWrapper.Message.Res
	if strings.Index(res, constants.ResourceTypeFiles) == 0 {
		isGranted = true
		return
	}

	// if key adapter doesn't override the permissions, check for user permissions
	var permissions map[string]bool
	var roles []string
	user, err = getUser(requestWrapper)
	if err != nil {
		return
	}

	roles, err = getRolesOfUser(user)
	if err != nil {
		return
	}

	if strings.Count(res, "/") == 1 {
		permissions, err = getPermissionsOnResources(roles, requestWrapper)
	} else if strings.Count(res, "/") == 2 {
		id := res[strings.LastIndex(res, "/") + 1:]
		permissions, err = getPermissionsOnObject(collection, id, roles)
	}

	for k, _ := range commandPermissionMap[requestWrapper.Message.Command] {
		if permissions[k] {
			isGranted = true
			break
		}
	}

	return
}

func getUser(requestWrapper messages.RequestWrapper) (user map[string]interface{}, err *utils.Error) {

	var userDataFromToken map[string]interface{}
	userDataFromToken, err = extractUserFromRequest(requestWrapper)

	if err != nil {
		return
	}

	if userDataFromToken != nil {
		userId := userDataFromToken["userId"].(string)
		user, err = database.Adapter.Get(constants.ClassUsers, userId)
		if err != nil {
			return
		}
	}

	return
}

func getRolesOfUser(user map[string]interface{}) (roles []string, err *utils.Error) {

	// TODO: get roles recursively

	if user != nil && user["_roles"] != nil {
		for _, r := range user["_roles"].([]interface{}) {
			roles = append(roles, "role:" + r.(string))
		}
		roles = append(roles, "user:" + user["_id"].(string))
	}
	roles = append(roles, "*")

	return
}

func extractUserFromRequest(requestWrapper messages.RequestWrapper) (user map[string]interface{}, err *utils.Error) {

	authHeaders := requestWrapper.Message.Headers["Authorization"]
	if authHeaders != nil && len(authHeaders) > 0 {
		accessToken := authHeaders[0]
		if strings.Index(accessToken, "Bearer ") != 0 {
			err = &utils.Error{http.StatusBadRequest, "Authorization header must start with 'Bearer ' prefix."}
			return
		}
		accessToken = accessToken[len("Bearer "):]
		user, err = verifyToken(accessToken)
	}
	return
}

func getPermissionsOnObject(collection string, id string, roles []string) (permissions map[string]bool, err *utils.Error) {

	var model map[string]interface{}
	model, err = database.Adapter.Get(collection, id)
	if err != nil {
		return
	}

	acl := model["_acl"]
	if acl != nil {
		permissions = make(map[string]bool)

		for _, v := range roles {
			p := acl.(map[string]interface{})[v]
			if p != nil {
				for kAcl, _ := range p.(map[string]interface{}) {
					permissions[kAcl] = true
				}
			}
		}
	} else {
		permissions = map[string]bool{
			"get": true,
			"update": true,
			"delete": true,
		}
	}

	return
}

func getPermissionsOnResources(roles []string, requestWrapper messages.RequestWrapper) (permissions map[string]bool, err *utils.Error) {

	// TODO get class type permissions and return them
	permissions = map[string]bool{
		"create": true,
		"query": true,
	}

	return
}

func verifyToken(tokenString string) (userData map[string]interface{}, err *utils.Error) {

	token, tokenErr := jwt.Parse(tokenString, func(t *jwt.Token) (interface{}, error) {
		return []byte("SIGN_IN_KEY"), nil
	})

	if tokenErr != nil {
		err = &utils.Error{http.StatusInternalServerError, "Parsing token failed."}
		return
	}

	if !token.Valid {
		err = &utils.Error{http.StatusUnauthorized, "Token is not valid."}
		return
	}

	mapClaims := token.Claims.(jwt.MapClaims)
	userData = mapClaims["user"].(map[string]interface{})
	return
}

var GenerateToken = func(userId string, userData map[string]interface{}) (tokenString string, err *utils.Error) {

	token := jwt.New(jwt.SigningMethodHS256)

	userTokenData := make(map[string]interface{})
	userTokenData["userId"] = userId

	if username, hasUsername := userData["username"]; hasUsername && username != "" {
		userTokenData["username"] = username
	}
	if email, hasEmail := userData["email"]; hasEmail && email != "" {
		userTokenData["email"] = email
	}

	mapClaims := token.Claims.(jwt.MapClaims)
	mapClaims["ver"] = "0.1"
	mapClaims["exp"] = time.Now().Add(time.Hour * 72).Unix()
	mapClaims["user"] = userTokenData

	var signErr error
	tokenString, signErr = token.SignedString([]byte("SIGN_IN_KEY"))
	if signErr != nil {
		err = &utils.Error{http.StatusInternalServerError, "Generating token failed. Reason: " + signErr.Error()}
	}
	return
}

