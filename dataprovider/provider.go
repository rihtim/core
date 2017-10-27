package dataprovider

import (
	"io"
	"github.com/rihtim/core/utils"
)

type Provider interface {
	Connect() (err *utils.Error)
	Create(collection string, data map[string]interface{}) (response map[string]interface{}, err *utils.Error)
	Get(collection string, id string) (response map[string]interface{}, err *utils.Error)
	Query(collection string, parameters map[string][]string) (response map[string]interface{}, err *utils.Error)
	Update(collection string, id string, data map[string]interface{}) (response map[string]interface{}, err *utils.Error)
	Delete(collection string, id string) (response map[string]interface{}, err *utils.Error)
	CreateFile(data io.ReadCloser) (response map[string]interface{}, err *utils.Error)
	GetFile(id string) (response []byte, err *utils.Error)
}
