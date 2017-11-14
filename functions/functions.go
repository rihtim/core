package functions

import (
	"github.com/rihtim/core/utils"
	"github.com/rihtim/core/messages"
	"github.com/rihtim/core/requestscope"
	"github.com/rihtim/core/dataprovider"
)

type FunctionHandler func(req messages.Message, rs requestscope.RequestScope, dp dataprovider.Provider) (resp messages.Message, editedRs requestscope.RequestScope, err *utils.Error)

type FunctionController interface {
	Contains(path string) bool
	Add(path string, handler FunctionHandler)
	Get(path string) (handler FunctionHandler, regex string)
	Execute(req messages.Message, rs requestscope.RequestScope, db dataprovider.Provider) (res messages.Message, editedRs requestscope.RequestScope, err *utils.Error)
}
