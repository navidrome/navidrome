package api_test

import (
	"encoding/xml"
	"github.com/deluan/gosonic/api/responses"
	. "github.com/smartystreets/goconvey/convey"
	"testing"
	"github.com/deluan/gosonic/tests"
)

func TestPing(t *testing.T) {
	tests.Init(t, false)

	_, w := Get(AddParams("/rest/ping.view"), "TestPing")

	Convey("Subject: Ping Endpoint\n", t, func() {
		Convey("Status code should be 200", func() {
			So(w.Code, ShouldEqual, 200)
		})
		Convey("The result should not be empty", func() {
			So(w.Body.Len(), ShouldBeGreaterThan, 0)
		})
		Convey("The result should be a valid ping response", func() {
			v := responses.Subsonic{}
			xml.Unmarshal(w.Body.Bytes(), &v)
			So(v.Status, ShouldEqual, "ok")
			So(v.Version, ShouldEqual, "1.0.0")
		})

	})
}
