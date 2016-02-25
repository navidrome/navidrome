package api_test

import (
	"testing"
	"encoding/xml"
	_ "github.com/deluan/gosonic/routers"
	. "github.com/smartystreets/goconvey/convey"
	"github.com/deluan/gosonic/controllers/responses"
	. "github.com/deluan/gosonic/tests"
)

func TestPing(t *testing.T) {
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

