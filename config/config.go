package config

import (
	"net/http"
	"io/ioutil"
	"encoding/json"
	"github.com/rihtim/core/utils"
	"github.com/Sirupsen/logrus"
)

var log = logrus.New()

func ReadConfig(fileName string) (configuration map[string]interface{}, err *utils.Error) {

	if len(fileName) == 0 {
		fileName = "rihtim-config.json"
	}

	configInBytes, configErr := ioutil.ReadFile(fileName)
	if configErr == nil {
		configParseErr := json.Unmarshal(configInBytes, &configuration)
		if configParseErr != nil {
			err = &utils.Error{http.StatusInternalServerError, "Parsing configuration file failed."};
			return
		}
	} else {
		err = &utils.Error{http.StatusInternalServerError, "No configuration file found with name '" + fileName + "'"};
		return
	}

	// check for port and set default value if not exists
	if configuration["port"] == nil {
		configuration["port"] = "1707"
		log.Info("Port is not defined. Setting to default port: 1707.")
	}

	return
}