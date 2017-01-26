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
	"github.com/rihtim/core/requestscope"
)

var HeaderKeyMaster = "Master-Key"
var KeyMaster = "master"

type BasicKeyAdapter struct {
	Keys map[string]string
}

func (ka *BasicKeyAdapter) Init(config map[string]interface{}) (err *utils.Error) {

	ka.Keys = make(map[string]string)

	// generate master key if not provided in config
	var hasMasterKey bool
	ka.Keys[KeyMaster], hasMasterKey = config[KeyMaster].(string)
	if !hasMasterKey {
		ka.Keys[KeyMaster], err = generateKey()
		if err != nil {return}
		log.Info("Master key generated: " + ka.Keys[KeyMaster])
	}

	keysJson, _ := json.Marshal(ka.Keys)
	ioutil.WriteFile("keys.json", keysJson, 0644)

	return
}

func (ka BasicKeyAdapter) IsKeyValid(keyName, key string) (bool) {
	return strings.EqualFold(ka.Keys[keyName], key)
}

func (ka BasicKeyAdapter) CheckKeyPermissions(headers map[string][]string) (isGrantedByKey bool, err *utils.Error) {

	masterKeys, hasMasterKey := headers[HeaderKeyMaster]
	if !hasMasterKey {
		return
	}

	if !keys.Adapter.IsKeyValid(KeyMaster, masterKeys[0]) {
		err = &utils.Error{http.StatusUnauthorized, "Master key is not valid."}
		return
	}

	isGrantedByKey = true

	masterKey := masterKeys[0]
	log.WithFields(logrus.Fields{
		"masterKey": masterKey[:10] + "..." + masterKey[len(masterKey)-10:len(masterKey)],
	}).Warning("Request contains a valid master key.")

	return
}

func generateKey() (string, *utils.Error) {

	key := make([]byte, 32)
	_, err := rand.Read(key)
	if err != nil {
		return "", &utils.Error{http.StatusInternalServerError, "Generating key failed."}
	}
	return fmt.Sprintf("%x", key), nil
}

//var RequireMasterKey = func(user map[string]interface{}, request, response messages.Message) (editedRequest, editedResponse messages.Message, err *utils.Error) {
var RequireMasterKey = func(requestScope requestscope.RequestScope, request, response messages.Message) (editedRequest, editedResponse messages.Message, err *utils.Error) {

	masterKeys, hasMasterKey := request.Headers[HeaderKeyMaster]
	if !hasMasterKey || !keys.Adapter.IsKeyValid(KeyMaster, masterKeys[0]) {
		err = &utils.Error{http.StatusForbidden, http.StatusText(http.StatusForbidden)}
		return
	}

	masterKey := masterKeys[0]
	log.WithFields(logrus.Fields{
		"masterKey": masterKey[:10] + "..." + masterKey[len(masterKey)-10:len(masterKey)],
	}).Warning("Request contains a valid master key.")

	editedRequest = request
	editedResponse = response
	return
}