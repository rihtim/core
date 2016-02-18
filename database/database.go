package database

import (
	"github.com/rihtim/core/utils"
	"io"
)

type DatabaseAdapter interface {
	Init(config map[string]interface{}) (err *utils.Error)
	Connect() (err *utils.Error)
	Create(collection string, data map[string]interface{}) (response map[string]interface{}, hookBody map[string]interface{}, err *utils.Error)
	Get(collection string, id string) (response map[string]interface{}, err *utils.Error)
	Query(collection string, parameters map[string][]string) (response map[string]interface{}, err *utils.Error)
	Update(collection string, id string, data map[string]interface{}) (response map[string]interface{}, hookBody map[string]interface{}, err *utils.Error)
	Delete(collection string, id string) (response map[string]interface{}, err *utils.Error)
	CreateFile(data io.ReadCloser) (response map[string]interface{}, hookBody map[string]interface{}, err *utils.Error)
	GetFile(id string) (response []byte, err *utils.Error)
}

var Adapter DatabaseAdapter
