package api_test

import (
	"encoding/xml"
	"github.com/deluan/gosonic/api/responses"
	. "github.com/smartystreets/goconvey/convey"
	"testing"
	"github.com/deluan/gosonic/tests"
)

func TestCheckParams(t *testing.T) {
	tests.Init(t, false)

	_, w := Get("/rest/ping.view", "TestCheckParams")

	Convey("Subject: CheckParams\n", t, func() {
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

	Convey("Subject: Authentication\n", t, func() {
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
