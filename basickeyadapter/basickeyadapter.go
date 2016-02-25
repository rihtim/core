package basickeyadapter

import (
	"fmt"
	"strings"
	"net/http"
	"io/ioutil"
	"crypto/rand"
	"encoding/json"
	"github.com/rihtim/core/log"
	"github.com/rihtim/core/utils"
	"github.com/rihtim/core/messages"
	"github.com/rihtim/core/keys"
	"github.com/Sirupsen/logrus"
)

var HeaderKeyMaster = "Master-Key"
var KeyMaster = "master"

type BasicKeyAdapter struct {
	keys map[string]string
}

func (ka *BasicKeyAdapter) Init(config map[string]interface{}) (err *utils.Error) {

	ka.keys = make(map[string]string)

	// generate master key if not provided in config
	var hasMasterKey bool
	ka.keys[KeyMaster], hasMasterKey = config[KeyMaster].(string)
	if !hasMasterKey {
		ka.keys[KeyMaster], err = generateKey()
		if err != nil {return}
		log.Info("Master key generated: " + ka.keys[KeyMaster])
	}

	keysJson, _ := json.Marshal(ka.keys)
	ioutil.WriteFile("keys.json", keysJson, 0644)

	return
}

func (ka BasicKeyAdapter) IsKeyValid(keyName, key string) (bool) {
	return strings.EqualFold(ka.keys[keyName], key)
}

func generateKey() (string, *utils.Error) {

	key := make([]byte, 32)
	_, err := rand.Read(key)
	if err != nil {
		return "", &utils.Error{http.StatusInternalServerError, "Generating key failed."}
	}
	return fmt.Sprintf("%x", key), nil
}

var RequireMasterKey = func(user map[string]interface{}, message messages.Message) (response messages.Message, err *utils.Error) {

	masterKeys, hasMasterKey := message.Headers[HeaderKeyMaster]
	if !hasMasterKey || !keys.Adapter.IsKeyValid(KeyMaster, masterKeys[0]) {
		err = &utils.Error{http.StatusForbidden, http.StatusText(http.StatusForbidden)}
		return
	}
	log.WithFields(logrus.Fields{
		"masterKey": masterKeys[0],
	}).Warning("Request contains a valid master key.")
	response = message
	return
}