package modifier

import (
	"testing"
	. "github.com/smartystreets/goconvey/convey"
	"net/http"
)

func TestExpandArray(t *testing.T) {

	Convey("Should validate expand config", t, func() {

		var originalFunction = isValidExpandConfig

		var isCalled bool
		isValidExpandConfig = func(config string) bool {
			isCalled = true
			return false
		}

		_, err := ExpandArray(nil, "")
		So(isCalled, ShouldBeTrue)
		So(err.Code, ShouldEqual, http.StatusBadRequest)

		// revert function to original
		isValidExpandConfig = originalFunction

	})

	Convey("Should return nil data error", t, func() {

		var originalFunction = isValidExpandConfig

		isValidExpandConfig = func(config string) bool {
			return true
		}

		_, err := ExpandArray(nil, "")
		So(err.Code, ShouldEqual, http.StatusInternalServerError)

		// revert function to original
		isValidExpandConfig = originalFunction
	})

	Convey("Should return nil 'data' field error", t, func() {

		var originalFunction = isValidExpandConfig

		isValidExpandConfig = func(config string) bool {
			return true
		}

		_, err := ExpandArray(make(map[string]interface{}), "")
		So(err.Code, ShouldEqual, http.StatusInternalServerError)

		// revert function to original
		isValidExpandConfig = originalFunction
	})

	Convey("Should call expand item for each item", t, func() {
		// TODO ExpandItem is not a variable. variables get error when used as recursive.
	})

}

func TestHelperMethods(t *testing.T) {

	Convey("Should seperate fields right", t, func() {

		config := "a,b"
		fields := seperateFields(config)
		So(len(fields), ShouldEqual, 2)
		So(fields[0], ShouldEqual, "a")
		So(fields[1], ShouldEqual, "b")

		config = "a(b),c,d"
		fields = seperateFields(config)
		So(len(fields), ShouldEqual, 3)
		So(fields[0], ShouldEqual, "a(b)")
		So(fields[1], ShouldEqual, "c")
		So(fields[2], ShouldEqual, "d")

		config = "a(b),c(d),e,f"
		fields = seperateFields(config)
		So(len(fields), ShouldEqual, 4)
		So(fields[0], ShouldEqual, "a(b)")
		So(fields[1], ShouldEqual, "c(d)")
		So(fields[2], ShouldEqual, "e")
		So(fields[3], ShouldEqual, "f")

		config = "a(b,c),d(e,f),g(h),i(j,k),l"
		fields = seperateFields(config)
		So(len(fields), ShouldEqual, 5)
		So(fields[0], ShouldEqual, "a(b,c)")
		So(fields[1], ShouldEqual, "d(e,f)")
		So(fields[2], ShouldEqual, "g(h)")
		So(fields[3], ShouldEqual, "i(j,k)")
		So(fields[4], ShouldEqual, "l")

		config = "a(b,c(d)),e(f),g(h(i,j)),k"
		fields = seperateFields(config)
		So(len(fields), ShouldEqual, 4)
		So(fields[0], ShouldEqual, "a(b,c(d))")
		So(fields[1], ShouldEqual, "e(f)")
		So(fields[2], ShouldEqual, "g(h(i,j))")
		So(fields[3], ShouldEqual, "k")

		config = "a(b,c(d),e),f(g(h(i(j)))),k,l,m"
		fields = seperateFields(config)
		So(len(fields), ShouldEqual, 5)
		So(fields[0], ShouldEqual, "a(b,c(d),e)")
		So(fields[1], ShouldEqual, "f(g(h(i(j))))")
		So(fields[2], ShouldEqual, "k")
		So(fields[3], ShouldEqual, "l")
		So(fields[4], ShouldEqual, "m")
	})

	Convey("Should return child's field and subfields", t, func() {

		config := "a"
		field, subFields := getChildFieldAndSubFields(config)
		So(field, ShouldEqual, "a")
		So(subFields, ShouldEqual, "")

		config = "a(b)"
		field, subFields = getChildFieldAndSubFields(config)
		So(field, ShouldEqual, "a")
		So(subFields, ShouldEqual, "b")

		config = "a(b(c))"
		field, subFields = getChildFieldAndSubFields(config)
		So(field, ShouldEqual, "a")
		So(subFields, ShouldEqual, "b(c)")

		config = "a(b,c(d),e)"
		field, subFields = getChildFieldAndSubFields(config)
		So(field, ShouldEqual, "a")
		So(subFields, ShouldEqual, "b,c(d),e")

	})

	Convey("Should check expand config", t, func() {

		So(isValidExpandConfig("a"), ShouldBeTrue)
		So(isValidExpandConfig("a("), ShouldBeFalse)
		So(isValidExpandConfig("a(b)"), ShouldBeTrue)

	})
}
