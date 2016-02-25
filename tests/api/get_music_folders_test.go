package api_test

import (
	_ "github.com/deluan/gosonic/routers"
	. "github.com/deluan/gosonic/tests"
	"testing"

	"encoding/xml"
	. "github.com/smartystreets/goconvey/convey"
)

func TestGetMusicFolders(t *testing.T) {
	_, w := Get(AddParams("/rest/getMusicFolders.view"), "TestGetMusicFolders")

	Convey("Subject: GetMusicFolders Endpoint\n", t, func() {
		Convey("Status code should be 200", func() {
			So(w.Code, ShouldEqual, 200)
		})
		Convey("The response should include the default folder", func() {
			v := new(string)
			err := xml.Unmarshal(w.Body.Bytes(), &v)
			So(err, ShouldBeNil)
			So(w.Body.String(), ShouldContainSubstring, `musicFolder id="0" name="iTunes Library"`)
		})
	})
}
