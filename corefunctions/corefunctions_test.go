package corefunctions

import (
	"testing"
	"net/http"
	"github.com/rihtim/core/messages"
	. "github.com/smartystreets/goconvey/convey"
	"github.com/eluleci/rihtim/core/constants"
	"github.com/rihtim/core/database"
	"github.com/rihtim/core/utils"
	"io"
	"strings"
)

type TestAdapter struct {
}

func (ma *TestAdapter) Init(config map[string]interface{}) (err *utils.Error) {
	return
}

func (ma *TestAdapter) Connect() (err *utils.Error) {
	return
}

func (ma TestAdapter) Create(collection string, data map[string]interface{}) (response map[string]interface{}, hookBody map[string]interface{}, err *utils.Error) {
	return
}

func (ma TestAdapter) Get(collection string, id string) (response map[string]interface{}, err *utils.Error) {
	response = make(map[string]interface{})

	if strings.EqualFold(id, "123") {
		response[constants.RolesIdentifier] = []interface{}{"role1"}
	}
	return
}

func (ma TestAdapter) Query(collection string, parameters map[string][]string) (response map[string]interface{}, err *utils.Error) {
	return
}

func (ma TestAdapter) Update(collection string, id string, data map[string]interface{}) (response map[string]interface{}, hookBody map[string]interface{}, err *utils.Error) {
	response = map[string]interface{}{"updatedAt":1}
	hookBody = map[string]interface{}{constants.IdIdentifier:id}
	for k, v := range data {
		hookBody[k] = v
	}
	return
}

func (ma TestAdapter) Delete(collection string, id string) (response map[string]interface{}, err *utils.Error) {
	return
}

func (ma TestAdapter) CreateFile(data io.ReadCloser) (response map[string]interface{}, hookBody map[string]interface{}, err *utils.Error) {
	return
}

func (ma TestAdapter) GetFile(id string) (response []byte, err *utils.Error) {
	return
}



func TestGrantRole(t *testing.T) {

	Convey("Should return bad request for wrong res format", t, func() {

		user := make(map[string]interface{})
		message := messages.Message{}
		message.Res = "/users"

		_, _, err := GrantRole(user, message)
		So(err.Code, ShouldEqual, http.StatusBadRequest)
	})

	Convey("Should return bad request for wrong collection", t, func() {

		user := make(map[string]interface{})
		message := messages.Message{}
		message.Res = "/objects/123/grantRole"

		_, _, err := GrantRole(user, message)
		So(err.Code, ShouldEqual, http.StatusBadRequest)
	})

	Convey("Should return bad request for nil body", t, func() {

		user := make(map[string]interface{})
		message := messages.Message{}
		message.Res = "/users/123/grantRole"

		_, _, err := GrantRole(user, message)
		So(err.Code, ShouldEqual, http.StatusBadRequest)
	})

	Convey("Should return bad request for empty _roles in body", t, func() {

		user := make(map[string]interface{})
		message := messages.Message{}
		message.Res = "/users/123/grantRole"
		message.Body = make(map[string]interface{})
		message.Body[constants.IdIdentifier] = "someId"

		_, _, err := GrantRole(user, message)
		So(err.Code, ShouldEqual, http.StatusBadRequest)
	})

	Convey("Should return bad request for request owners lack of role info", t, func() {

		user := make(map[string]interface{})
		message := messages.Message{}
		message.Res = "/users/123/grantRole"
		message.Body = make(map[string]interface{})
		message.Body[constants.RolesIdentifier] = make([]string, 0)

		_, _, err := GrantRole(user, message)
		So(err.Code, ShouldEqual, http.StatusUnauthorized)
	})

	Convey("Should return bad request for request owners lack of permissions", t, func() {

		requestOwnersRoles := []interface{}{"role1"}
		user := make(map[string]interface{})
		user[constants.RolesIdentifier] = requestOwnersRoles

		message := messages.Message{}
		message.Res = "/users/123/grantRole"
		message.Body = make(map[string]interface{})
		rolesToGrant := []interface{}{"role2"}
		message.Body[constants.RolesIdentifier] = rolesToGrant

		_, _, err := GrantRole(user, message)
		So(err.Code, ShouldEqual, http.StatusUnauthorized)
	})

	Convey("Should return bad request for request owners lack of permissions", t, func() {

		requestOwnersRoles := []interface{}{"role1"}
		user := make(map[string]interface{})
		user[constants.RolesIdentifier] = requestOwnersRoles

		message := messages.Message{}
		message.Res = "/users/123/grantRole"
		message.Body = make(map[string]interface{})
		rolesToGrant := []interface{}{"role1", "role2"}
		message.Body[constants.RolesIdentifier] = rolesToGrant

		_, _, err := GrantRole(user, message)
		So(err.Code, ShouldEqual, http.StatusUnauthorized)
	})

	Convey("Should update roles.", t, func() {

		database.Adapter = new(TestAdapter)

		requestOwnersRoles := []interface{}{"role2", "role3"}
		user := make(map[string]interface{})
		user[constants.RolesIdentifier] = requestOwnersRoles

		message := messages.Message{}
		message.Res = "/users/123/grantRole"
		message.Body = make(map[string]interface{})
		rolesToGrant := []interface{}{"role3"}
		message.Body[constants.RolesIdentifier] = rolesToGrant

		response, hookBody, err := GrantRole(user, message)

		So(err, ShouldBeNil)
		So(response, ShouldNotBeNil)
		So(hookBody, ShouldNotBeNil)
	})
}
