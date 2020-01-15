package scanner_legacy

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestExtractLocation(t *testing.T) {

	Convey("Given a path with a plus (+) signal", t, func() {
		location := "file:///Users/deluan/Music/iTunes%201/iTunes%20Media/Music/Chance/Six%20Through%20Ten/03%20Forgive+Forget.m4a"

		Convey("When I decode it", func() {
			path := extractPath(location)

			Convey("I get the correct path", func() {
				So(path, ShouldEqual, "/Users/deluan/Music/iTunes 1/iTunes Media/Music/Chance/Six Through Ten/03 Forgive+Forget.m4a")
			})

		})

	})

}
