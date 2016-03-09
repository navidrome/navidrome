package utils

import (
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
)

func TestTimeConversion(t *testing.T) {

	Convey("Conversion should work both ways", t, func() {
		now := time.Date(2002, 8, 9, 12, 11, 13, 1000000, time.Local)
		milli := ToMillis(now)
		So(ToTime(milli).String(), ShouldEqual, now.String())
	})
}
