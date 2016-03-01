package scanner

import (
	"testing"
	. "github.com/smartystreets/goconvey/convey"
	"github.com/deluan/gosonic/utils"
	"github.com/deluan/gosonic/models"
"github.com/deluan/gosonic/tests"
)

func TestCollectIndex(t *testing.T) {
	tests.Init(t, false)

	ig := utils.IndexGroups{"A":"A", "B":"B", "Tom":"Tom", "X":"X-Z"}

	Convey("Simple Name", t, func() {
		a := &models.Artist{Name: "Björk"}
		artistIndex := make(map[string]tempIndex)

		collectIndex(ig, a, artistIndex)

		So(artistIndex, ShouldContainKey, "B")
		So(artistIndex["B"], ShouldContainKey, "björk")

		for _, k := range []string{"A", "Tom", "X-Z", "#"} {
			So(artistIndex, ShouldNotContainKey, k)
		}
	})

	Convey("Name not in the index", t, func() {
		a := &models.Artist{Name: "Kraftwerk"}
		artistIndex := make(map[string]tempIndex)

		collectIndex(ig, a, artistIndex)

		So(artistIndex, ShouldContainKey, "#")
		So(artistIndex["#"], ShouldContainKey, "kraftwerk")

		for _, k := range []string{"A", "B", "Tom", "X-Z"} {
			So(artistIndex, ShouldNotContainKey, k)
		}
	})

	Convey("Name starts with an article", t, func() {
		a := &models.Artist{Name: "The The"}
		artistIndex := make(map[string]tempIndex)

		collectIndex(ig, a, artistIndex)

		So(artistIndex, ShouldContainKey, "#")
		So(artistIndex["#"], ShouldContainKey, "the")

		for _, k := range []string{"A", "B", "Tom", "X-Z"} {
			So(artistIndex, ShouldNotContainKey, k)
		}
	})

	Convey("Name match a multichar entry", t, func() {
		a := &models.Artist{Name: "Tom Waits"}
		artistIndex := make(map[string]tempIndex)

		collectIndex(ig, a, artistIndex)

		So(artistIndex, ShouldContainKey, "Tom")
		So(artistIndex["Tom"], ShouldContainKey, "tom waits")

		for _, k := range []string{"A", "B", "X-Z", "#"} {
			So(artistIndex, ShouldNotContainKey, k)
		}
	})
}
