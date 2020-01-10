package engine_test

import (
	"bytes"
	"image"
	"testing"

	"github.com/cloudsonic/sonic-server/domain"
	"github.com/cloudsonic/sonic-server/engine"
	"github.com/cloudsonic/sonic-server/persistence"
	. "github.com/cloudsonic/sonic-server/tests"
	. "github.com/smartystreets/goconvey/convey"
)

func TestCover(t *testing.T) {
	Init(t, false)

	mockMediaFileRepo := persistence.CreateMockMediaFileRepo()
	mockAlbumRepo := persistence.CreateMockAlbumRepo()

	cover := engine.NewCover(mockMediaFileRepo, mockAlbumRepo)
	out := new(bytes.Buffer)

	Convey("Subject: GetCoverArt Endpoint", t, func() {
		Convey("When id is not found", func() {
			mockMediaFileRepo.SetData(`[]`, 1)
			err := cover.Get("1", 0, out)

			Convey("Then return default cover", func() {
				So(err, ShouldBeNil)
				So(out.Bytes(), ShouldMatchMD5, "963552b04e87a5a55e993f98a0fbdf82")
			})
		})
		Convey("When id is found", func() {
			mockMediaFileRepo.SetData(`[{"ID":"2","HasCoverArt":true,"Path":"tests/fixtures/01 Invisible (RED) Edit Version.mp3"}]`, 1)
			err := cover.Get("2", 0, out)

			Convey("Then it should return the cover from the file", func() {
				So(err, ShouldBeNil)
				So(out.Bytes(), ShouldMatchMD5, "e859a71cd1b1aaeb1ad437d85b306668")
			})
		})
		Convey("When there is an error accessing the database", func() {
			mockMediaFileRepo.SetData(`[{"ID":"2","HasCoverArt":true,"Path":"tests/fixtures/01 Invisible (RED) Edit Version.mp3"}]`, 1)
			mockMediaFileRepo.SetError(true)
			err := cover.Get("2", 0, out)

			Convey("Then error should not be nil", func() {
				So(err, ShouldNotBeNil)
			})
		})
		Convey("When id is found but file is not present", func() {
			mockMediaFileRepo.SetData(`[{"ID":"2","HasCoverArt":true,"Path":"tests/fixtures/NOT_FOUND.mp3"}]`, 1)
			err := cover.Get("2", 0, out)

			Convey("Then it should return DatNotFound error", func() {
				So(err, ShouldEqual, domain.ErrNotFound)
			})
		})
		Convey("When specifying a size", func() {
			mockMediaFileRepo.SetData(`[{"ID":"2","HasCoverArt":true,"Path":"tests/fixtures/01 Invisible (RED) Edit Version.mp3"}]`, 1)
			err := cover.Get("2", 100, out)

			Convey("Then image returned should be 100x100", func() {
				So(err, ShouldBeNil)
				So(out.Bytes(), ShouldMatchMD5, "04378f523ca3e8ead33bf7140d39799e")
				img, _, err := image.Decode(bytes.NewReader(out.Bytes()))
				So(err, ShouldBeNil)
				So(img.Bounds().Max.X, ShouldEqual, 100)
				So(img.Bounds().Max.Y, ShouldEqual, 100)
			})
		})
		Convey("When id is for an album", func() {
			mockAlbumRepo.SetData(`[{"ID":"1","CoverArtPath":"tests/fixtures/01 Invisible (RED) Edit Version.mp3"}]`, 1)
			err := cover.Get("al-1", 0, out)

			Convey("Then it should return the cover for the album", func() {
				So(err, ShouldBeNil)
				So(out.Bytes(), ShouldMatchMD5, "e859a71cd1b1aaeb1ad437d85b306668")
			})
		})

		Reset(func() {
			mockMediaFileRepo.SetData("[]", 0)
			mockMediaFileRepo.SetError(false)
			out = new(bytes.Buffer)
		})
	})
}
