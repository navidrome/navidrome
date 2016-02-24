package test

import (
	"testing"
	"encoding/xml"
	_ "github.com/deluan/gosonic/routers"
	. "github.com/deluan/gosonic/tests"
	. "github.com/smartystreets/goconvey/convey"
	"github.com/deluan/gosonic/controllers/responses"
)

func TestCheckParams(t *testing.T) {
	_, w := Get("/rest/ping.view", "TestCheckParams")

	Convey("Subject: Validation\n", t, func() {
		Convey("Status code should be 200", func() {
			So(w.Code, ShouldEqual, 200)
		})
		Convey("The errorCode should be 10", func() {
			So(w.Body.String(), ShouldContainSubstring, `error code="10" message=`)
		})
		Convey("The status should be 'fail'", func() {
			v := responses.Subsonic{}
			xml.Unmarshal(w.Body.Bytes(), &v)
			So(v.Status, ShouldEqual, "fail")
		})
	})
}

func TestAuthentication(t *testing.T) {
	_, w := Get("/rest/ping.view?u=INVALID&p=INVALID&c=test&v=1.0.0", "TestAuthentication")

	Convey("Subject: Validation\n", t, func() {
		Convey("Status code should be 200", func() {
			So(w.Code, ShouldEqual, 200)
		})
		Convey("The errorCode should be 10", func() {
			So(w.Body.String(), ShouldContainSubstring, `error code="40" message=`)
		})
		Convey("The status should be 'fail'", func() {
			v := responses.Subsonic{}
			xml.Unmarshal(w.Body.Bytes(), &v)
			So(v.Status, ShouldEqual, "fail")
		})
	})
}


