package auth

import (
	"github.com/rihtim/core/messages"
	"github.com/rihtim/core/utils"
)

type AuthAdapter interface {
	GetUser(request messages.Message) (user map[string]interface{}, err *utils.Error)
	GenerateAuthData(user map[string]interface{}) (authData map[string]interface{}, err *utils.Error)
	IsGranted(user interface{}, request messages.Message) (isGranted bool, err *utils.Error)
}

// var Adapter AuthAdapter