package controller_test

import (
	"testing"
	_ "github.com/deluan/gosonic/routers"
	. "github.com/deluan/gosonic/tests"
	"encoding/xml"
	. "github.com/smartystreets/goconvey/convey"
)

func TestGetLicense(t *testing.T) {
	_, w := Get(AddParams("/rest/getLicense.view"), "TestGetLicense")

	Convey("Subject: GetLicense Endpoint\n", t, func() {
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

