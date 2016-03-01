package validator

import (
	"testing"
	. "github.com/smartystreets/goconvey/convey"
)


func TestGrantRole(t *testing.T) {

	Convey("Should return error for containing fields", t, func() {

		input := map[string]interface{}{
			"a":"a",
			"b":"b",
			"c":"c",
		}

		restrictedFields := map[string]bool{"b":false}
		err := ValidateInputFields(restrictedFields, input)

		So(err, ShouldNotBeNil)
	})

	Convey("Should return error for not containing fields", t, func() {

		input := map[string]interface{}{
			"a":"a",
			"b":"b",
			"c":"c",
		}

		expectedFields := map[string]bool{"d":true}
		err := ValidateInputFields(expectedFields, input)

		So(err, ShouldNotBeNil)
	})

	Convey("Should return error for both containing and not containing fields", t, func() {

		input := map[string]interface{}{
			"a":"a",
			"b":"b",
			"c":"c",
		}

		fields := map[string]bool{"b": false, "d":true}
		err := ValidateInputFields(fields, input)

		So(err, ShouldNotBeNil)
	})

	Convey("Should not return error", t, func() {

		input := map[string]interface{}{
			"a":"a",
			"b":"b",
			"c":"c",
		}

		fields := map[string]bool{"a": true, "b":true, "c":true, "d":false, "e":false}
		err := ValidateInputFields(fields, input)

		So(err, ShouldBeNil)
	})
}
