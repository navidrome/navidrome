package test

import (
	"testing"
	_ "github.com/deluan/gosonic/routers"
	. "github.com/deluan/gosonic/tests"

	. "github.com/smartystreets/goconvey/convey"
)

func TestGetMusicFolders(t *testing.T) {
	_, w := Get(AddParams("/rest/getMusicFolders.view"), "TestGetMusicFolders")

	Convey("Subject: GetMusicFolders Endpoint\n", t, func() {
		Convey("Status code should be 200", func() {
			So(w.Code, ShouldEqual, 200)
		})
	})
}

