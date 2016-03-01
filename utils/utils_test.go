package utils

import (
	"testing"
	. "github.com/smartystreets/goconvey/convey"
)

func TestParseIndexGroup(t *testing.T) {

	Convey("Two simple entries", t, func() {
		parsed := ParseIndexGroups("A The")

		So(parsed, ShouldContainKey, "A")
		So(parsed["A"], ShouldEqual, "A")

		So(parsed, ShouldContainKey, "The")
		So(parsed["The"], ShouldEqual, "The")
	})

	Convey("An entry with a group", t, func() {
		parsed := ParseIndexGroups("A-C(ABC) Z")

		So(parsed, ShouldContainKey, "A")
		So(parsed["A"], ShouldEqual, "A-C")
		So(parsed, ShouldContainKey, "B")
		So(parsed["B"], ShouldEqual, "A-C")
		So(parsed, ShouldContainKey, "C")
		So(parsed["C"], ShouldEqual, "A-C")

		So(parsed["Z"], ShouldEqual, "Z")

	})
	Convey("Correctly parses UTF-8", t, func() {
		parsed := ParseIndexGroups("UTF8(宇A海)")

		So(parsed, ShouldContainKey, "宇")
		So(parsed["宇"], ShouldEqual, "UTF8")
		So(parsed, ShouldContainKey, "A")
		So(parsed["A"], ShouldEqual, "UTF8")
		So(parsed, ShouldContainKey, "海")
		So(parsed["海"], ShouldEqual, "UTF8")
	})
}