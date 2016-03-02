package api_test

import (
	. "github.com/deluan/gosonic/tests"
	. "github.com/smartystreets/goconvey/convey"
	"testing"
)

func TestGetLicense(t *testing.T) {
	Init(t, false)

	_, w := Get(AddParams("/rest/getLicense.view"), "TestGetLicense")

	Convey("Subject: GetLicense Endpoint", t, func() {
		Convey("Status code should be 200", func() {
			So(w.Code, ShouldEqual, 200)
		})
		Convey("The license should always be valid", func() {
			So(UnindentJSON(w.Body.Bytes()), ShouldContainSubstring, `"license":{"valid":true}`)
		})

	})
}
