package stream

import (
	. "github.com/deluan/gosonic/tests"
	. "github.com/smartystreets/goconvey/convey"
	"testing"
)

func TestDownsampling(t *testing.T) {

	Init(t, false)

	Convey("Subject: createDownsamplingCommand", t, func() {

		Convey("It should create a valid command line", func() {
			cmd, args := createDownsamplingCommand("/music library/file.mp3", 128)

			So(cmd, ShouldEqual, "ffmpeg")
			So(args[0], ShouldEqual, "-i")
			So(args[1], ShouldEqual, "/music library/file.mp3")
			So(args[2], ShouldEqual, "-b:a")
			So(args[3], ShouldEqual, "128k")
			So(args[4], ShouldEqual, "mp3")
			So(args[5], ShouldEqual, "-")
		})

	})

}
