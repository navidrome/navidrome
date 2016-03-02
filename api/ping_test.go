package api_test

import (
	"github.com/deluan/gosonic/api/responses"
	. "github.com/deluan/gosonic/tests"
	. "github.com/smartystreets/goconvey/convey"
	"testing"
	"encoding/json"
)

func TestPing(t *testing.T) {
	Init(t, false)

	_, w := Get(AddParams("/rest/ping.view"), "TestPing")

	Convey("Subject: Ping Endpoint", t, func() {
		Convey("Status code should be 200", func() {
			So(w.Code, ShouldEqual, 200)
		})
		Convey("The result should not be empty", func() {
			So(w.Body.Len(), ShouldBeGreaterThan, 0)
		})
		Convey("The result should be a valid ping response", func() {
			v := responses.JsonWrapper{}
			err := json.Unmarshal(w.Body.Bytes(), &v)
			So(err, ShouldBeNil)
			So(v.Subsonic.Status, ShouldEqual, "ok")
			So(v.Subsonic.Version, ShouldEqual, "1.0.0")
		})

	})
}
