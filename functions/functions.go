package functions

import (
	"github.com/rihtim/core/utils"
	"github.com/rihtim/core/messages"
	"github.com/rihtim/core/requestscope"
	"github.com/rihtim/core/dataprovider"
)

type FunctionHandler func(req messages.Message, rs requestscope.RequestScope, extras interface{}, dp dataprovider.Provider) (resp messages.Message, editedRs requestscope.RequestScope, err *utils.Error)

type FunctionController interface {
	FindIndex(path, method string) int
	Contains(path string) bool
	Add(path, method string, handler FunctionHandler, extras interface{})
	Execute(req messages.Message, rs requestscope.RequestScope, db dataprovider.Provider) (res messages.Message, editedRs requestscope.RequestScope, err *utils.Error)
}
