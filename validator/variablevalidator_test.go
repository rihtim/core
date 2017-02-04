package validator

import (
	"testing"
	"github.com/rihtim/core/messages"
	"github.com/rihtim/core/requestscope"
	. "github.com/smartystreets/goconvey/convey"
)

type NewUserRequest struct {
	Username string `json:"username" validate:"min=3,max=40,regexp=^[a-zA-Z]*$"`
	Name     string `json:"name" validate:"nonzero"`
	Age      int    `json:"age" validate:"min=21"`
	Password string `json:"password" validate:"min=8"`
}

func TestVariableValidation(t *testing.T) {

	Convey("Should return error for invalid variables", t, func() {

		request := messages.Message{
			Body: map[string]interface{}{
				"username": "someUserName",
				"name": "someUserName",
			},
		}

		_, _, _, err := VariableValidator(requestscope.RequestScope{}, &NewUserRequest{}, request, messages.Message{})

		So(err, ShouldNotBeNil)
	})
}
