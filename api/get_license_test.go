package api_test

import (
	"encoding/xml"
	"github.com/deluan/gosonic/tests"
	. "github.com/smartystreets/goconvey/convey"
	"testing"
)

func TestGetLicense(t *testing.T) {
	tests.Init(t, false)

	_, w := Get(AddParams("/rest/getLicense.view"), "TestGetLicense")

	Convey("Subject: GetLicense Endpoint", t, func() {
		Convey("Status code should be 200", func() {
			So(w.Code, ShouldEqual, 200)
		})
		Convey("The license should always be valid", func() {
			v := new(string)
			err := xml.Unmarshal(w.Body.Bytes(), &v)
			So(err, ShouldBeNil)
			So(w.Body.String(), ShouldContainSubstring, `license valid="true"`)
		})

	})
}
