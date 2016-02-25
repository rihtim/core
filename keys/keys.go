package keys

import (
	"github.com/rihtim/core/utils"
)

type KeyAdapter interface {
	Init(config map[string]interface{}) (err *utils.Error)
	IsKeyValid(keyName, key string) (bool)
}

var Adapter KeyAdapter
