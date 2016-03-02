package api_test

import (
	"testing"

	. "github.com/deluan/gosonic/tests"
	. "github.com/smartystreets/goconvey/convey"
)

func TestGetMusicFolders(t *testing.T) {
	Init(t, false)

	_, w := Get(AddParams("/rest/getMusicFolders.view"), "TestGetMusicFolders")

	Convey("Subject: GetMusicFolders Endpoint", t, func() {
		Convey("Status code should be 200", func() {
			So(w.Code, ShouldEqual, 200)
		})
		Convey("The response should include the default folder", func() {
			So(UnindentJSON(w.Body.Bytes()), ShouldContainSubstring, `{"musicFolder":[{"id":"0","name":"iTunes Library"}]}`)
		})
	})
}
