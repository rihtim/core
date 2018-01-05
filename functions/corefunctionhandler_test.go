package functions

import (
	"testing"
	"net/http"
	"github.com/rihtim/core/utils"
	"github.com/rihtim/core/messages"
	"github.com/rihtim/core/requestscope"
	"github.com/rihtim/core/dataprovider"
	. "github.com/smartystreets/goconvey/convey"
)

func TestImport(t *testing.T) {

	Convey("Given a core function controller", t, func() {
		coreFunctionController := CoreFunctionController{}

		Convey("When no function is added", func() {

			Convey("It should return contains false", func() {
				So(coreFunctionController.Contains("/{id}/convert", http.MethodPost), ShouldBeFalse)
			})

			Convey("Handler array length should be 0", func() {
				So(len(coreFunctionController.functionHandlers), ShouldEqual, 0)
			})

			Convey("FindIndex should return -1", func() {
				So(coreFunctionController.FindIndex("/{id}/convert", http.MethodPost), ShouldEqual, -1)
			})
		})

		Convey("When a function is added", func() {
			convertFunc := func(req messages.Message, rs requestscope.RequestScope, extras interface{}, dp dataprovider.Provider) (resp messages.Message, editedRs requestscope.RequestScope, err *utils.Error) {
				return
			}
			coreFunctionController.Add("/{id}/convert", http.MethodPost, convertFunc, nil)

			Convey("It should return contains true", func() {
				So(coreFunctionController.Contains("/{id}/convert", http.MethodPost), ShouldBeTrue)
			})

			Convey("Handler array length should be 1", func() {
				So(len(coreFunctionController.functionHandlers), ShouldEqual, 1)
			})

			Convey("FindIndex should return 0", func() {
				So(coreFunctionController.FindIndex("/{id}/convert", http.MethodPost), ShouldEqual, 0)
			})

			Convey("FindIndex for another method should return -1", func() {
				So(coreFunctionController.FindIndex("/{id}/convert", http.MethodGet), ShouldEqual, -1)
			})

			Convey("FindIndex for another path should return -1", func() {
				So(coreFunctionController.FindIndex("/{id}/view", http.MethodPost), ShouldEqual, -1)
			})
		})

		Convey("When a request is received", func() {

			isCalled := false
			rsKey := "someKey"
			rsValue := "someValue"
			var receivedRs requestscope.RequestScope

			convertFunc := func(req messages.Message, rs requestscope.RequestScope, extras interface{}, dp dataprovider.Provider) (resp messages.Message, editedRs requestscope.RequestScope, err *utils.Error) {
				isCalled = true
				receivedRs = rs
				editedRs = rs.Copy()

				// to see if original rs is changed
				rs.Set(rsKey, rsValue)

				// to see if edited rs contains value
				editedRs.Set(rsKey, rsValue)
				return
			}
			coreFunctionController.Add("/{id}/convert", http.MethodPost, convertFunc, nil)

			req := messages.Message{
				Res:     "/someUserId/convert",
				Command: http.MethodPost,
			}
			rs := requestscope.Init()

			_, editedRs, _ := coreFunctionController.Execute(req, rs, nil)

			Convey("Function should be called ", func() {
				So(isCalled, ShouldBeTrue)
			})

			Convey("Original request scope should not be changed", func() {
				So(rs.Contains(rsKey), ShouldBeFalse)
			})

			Convey("Input request scope should contain path parameter value", func() {
				So(receivedRs.IsEmpty(), ShouldBeFalse)
				So(receivedRs.Contains("id"), ShouldBeTrue)
				So(receivedRs.Get("id"), ShouldEqual, "someUserId")
			})

			Convey("Edited request scope should contain value set in function", func() {
				So(editedRs.IsEmpty(), ShouldBeFalse)
				So(editedRs.Contains(rsKey), ShouldBeTrue)
				So(receivedRs.Get(rsKey), ShouldEqual, rsValue)
			})
		})
	})
}
