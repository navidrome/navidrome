package api_test

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	"github.com/deluan/gosonic/tests"
)

func TestGetIndexes(t *testing.T) {
	tests.Init(t, false)

	_, w := Get(AddParams("/rest/getIndexes.view"), "TestGetIndexes")

	Convey("Subject: GetIndexes Endpoint", t, func() {
		Convey("Status code should be 200", func() {
			So(w.Code, ShouldEqual, 200)
		})
	})
}
