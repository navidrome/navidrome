package api_test

import (
	"testing"

	"github.com/deluan/gosonic/api/responses"
	"github.com/deluan/gosonic/domain"
	"github.com/deluan/gosonic/engine"
	"github.com/deluan/gosonic/persistence"
	. "github.com/deluan/gosonic/tests"
	"github.com/deluan/gosonic/utils"
	. "github.com/smartystreets/goconvey/convey"
)

func TestGetAlbumList(t *testing.T) {
	Init(t, false)

	mockAlbumRepo := persistence.CreateMockAlbumRepo()
	utils.DefineSingleton(new(domain.AlbumRepository), func() domain.AlbumRepository {
		return mockAlbumRepo
	})

	mockNowPlayingRepo := engine.CreateMockNowPlayingRepo()
	utils.DefineSingleton(new(engine.NowPlayingRepository), func() engine.NowPlayingRepository {
		return mockNowPlayingRepo
	})

	Convey("Subject: GetAlbumList Endpoint", t, func() {
		mockAlbumRepo.SetData(`[
				{"Id":"A","Name":"Vagarosa","ArtistId":"2"},
				{"Id":"C","Name":"Liberation: The Island Anthology","ArtistId":"3"},
				{"Id":"B","Name":"Planet Rock","ArtistId":"1"}]`, 1)

		Convey("Should fail if missing 'type' parameter", func() {
			_, w := Get(AddParams("/rest/getAlbumList.view"), "TestGetAlbumList")

			So(w.Body, ShouldReceiveError, responses.ERROR_MISSING_PARAMETER)
		})
		Convey("Return fail on Album Table error", func() {
			mockAlbumRepo.SetError(true)
			_, w := Get(AddParams("/rest/getAlbumList.view", "type=newest"), "TestGetAlbumList")

			So(w.Body, ShouldReceiveError, responses.ERROR_GENERIC)
		})
		Convey("Type is invalid", func() {
			_, w := Get(AddParams("/rest/getAlbumList.view", "type=not_implemented"), "TestGetAlbumList")

			So(w.Body, ShouldReceiveError, responses.ERROR_GENERIC)
		})
		Convey("Max size = 500", func() {
			_, w := Get(AddParams("/rest/getAlbumList.view", "type=newest", "size=501"), "TestGetAlbumList")
			So(w.Body, ShouldBeAValid, responses.AlbumList{})
			So(mockAlbumRepo.Options.Size, ShouldEqual, 500)
			So(mockAlbumRepo.Options.Alpha, ShouldBeTrue)
		})
		Convey("Type == newest", func() {
			_, w := Get(AddParams("/rest/getAlbumList.view", "type=newest"), "TestGetAlbumList")
			So(w.Body, ShouldBeAValid, responses.AlbumList{})
			So(mockAlbumRepo.Options.SortBy, ShouldEqual, "CreatedAt")
			So(mockAlbumRepo.Options.Desc, ShouldBeTrue)
			So(mockAlbumRepo.Options.Alpha, ShouldBeTrue)
		})
		Reset(func() {
			mockAlbumRepo.SetData("[]", 0)
			mockAlbumRepo.SetError(false)
		})
	})
}
