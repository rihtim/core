package validator

import (
	"github.com/rihtim/core/utils"
	"net/http"
)

/**
 * Checks the given input against the given field map. The fields map should contain the expected fields with 'true'
 * and the restricted fields with 'false' value.
 */
var ValidateInputFields = func(fields map[string]bool, data map[string]interface{}) (err *utils.Error) {

	for key, shouldContain := range fields {
		if _, containsField := data[key]; containsField != shouldContain {
			if shouldContain {
				err = &utils.Error{http.StatusBadRequest, "Input must contain '" + key + "' field."}
			} else {
				err = &utils.Error{http.StatusBadRequest, "Input cannot contain '" + key + "' field."}
			}
			return
		}
	}
	return
}
