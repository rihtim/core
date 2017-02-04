package validator

import (
	"github.com/rihtim/core/utils"
	"net/http"
	"github.com/rihtim/core/requestscope"
	"github.com/rihtim/core/messages"
)

var InputFieldValidator = func(requestScope requestscope.RequestScope, extras interface{}, request, response messages.Message) (editedRequest, editedResponse messages.Message, editedRequestScope requestscope.RequestScope, err *utils.Error) {

	if extras == nil {
		err = &utils.Error{http.StatusInternalServerError, "Input field validator expects 'extras' to be the expected/unexcpected field map, not nil."}
	}

	err = ValidateInputFields(extras.(map[string]bool), request.Body)
	return
}

var ExactInputFieldValidator = func(requestScope requestscope.RequestScope, extras interface{}, request, response messages.Message) (editedRequest, editedResponse messages.Message, editedRequestScope requestscope.RequestScope, err *utils.Error) {

	if extras == nil {
		err = &utils.Error{http.StatusInternalServerError, "Input field validator expects 'extras' to be the expected/unexcpected field map, not nil."}
	}

	err = ValidateExactInputFields(extras.(map[string]bool), request.Body)
	return
}

// Checks the given input against the given field map. The fields map should contain the expected
// fields defined with 'true' and shouldn't contain the restricted fields defined with 'false'.
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

// ValidateExactInputFields works like ValidateInputFields but additionally checks if is there any field in
// the data that is not specified in the field map.
var ValidateExactInputFields = func(fields map[string]bool, data map[string]interface{}) (err *utils.Error) {

	err = ValidateInputFields(fields, data)
	if err != nil {
		return
	}

	for key := range data {
		if _, containsKeyInFields := fields[key]; !containsKeyInFields {
			err = &utils.Error{http.StatusBadRequest, "Unexpected field '" + key + "'."}
		}
	}
	return
}
