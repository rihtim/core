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

type JWTAdapter struct {
	SignKey string
	Version string
	Expiration int32
}

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

func (a *JWTAdapter) GetUser(request messages.Message) (user map[string]interface{}, err *utils.Error) {

	var userDataFromToken map[string]interface{}
	userDataFromToken, err = extractUserFromRequest(request)

	if err != nil {
		return
	}

	if userDataFromToken != nil {
		userId := userDataFromToken["userId"].(string)
		user, err = database.Adapter.Get(constants.ClassUsers, userId)
	}
	return
}

func (a *JWTAdapter) GenerateAuthData(user map[string]interface{}) (authData map[string]interface{}, err *utils.Error) {

	token := jwt.New(jwt.SigningMethodHS256)

	mapClaims := token.Claims.(jwt.MapClaims)
	mapClaims["ver"] = a.Version
	mapClaims["exp"] = time.Now().Add(time.Hour *time.Duration(a.Expiration)).Unix()
	mapClaims["user"] = user

	tokenString, signErr := token.SignedString([]byte(a.SignKey))
	if signErr != nil {
		err = &utils.Error{http.StatusInternalServerError, "Generating token failed. Reason: " + signErr.Error()}
	}

	authData = map[string]interface{}{
		"token": tokenString,
	}
	return
}

func (a *JWTAdapter) IsGranted(user map[string]interface{}, request messages.Message) (isGranted bool, err *utils.Error) {

	// grant the request for everyone for file resources
	if strings.Index(request.Res, constants.ResourceTypeFiles) == 0 {
		isGranted = true
		return
	}

	regex := utils.ConvertRichUrlToRegex("/{collection}/", false)
	urlParams := utils.GetParamsFromRichUrl(regex, request.Res)
	collection := urlParams["collection"]

	// check for user permissions
	var roles []string
	var permissions map[string]bool

	roles, err = getRolesOfUser(user)
	if err != nil {
		return
	}

	if strings.Count(request.Res, "/") == 1 {
		permissions, err = getPermissionsOnResources(roles, request)
	} else if strings.Count(request.Res, "/") == 2 {
		id := request.Res[strings.LastIndex(request.Res, "/") + 1:]
		permissions, err = getPermissionsOnObject(collection, id, roles)
	}

	for k, _ := range commandPermissionMap[request.Command] {
		if permissions[k] {
			isGranted = true
			break
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

func extractUserFromRequest(request messages.Message) (user map[string]interface{}, err *utils.Error) {

	authHeaders := request.Headers["Authorization"]
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

func getPermissionsOnResources(roles []string, request messages.Message) (permissions map[string]bool, err *utils.Error) {

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
		err = &utils.Error{http.StatusInternalServerError, "Parsing token failed. Reason: " + tokenErr.Error()}
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