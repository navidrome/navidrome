package engine_test

import (
	"testing"
	"time"

	"github.com/deluan/gosonic/engine"
	"github.com/deluan/gosonic/itunesbridge"
	"github.com/deluan/gosonic/persistence"
	. "github.com/deluan/gosonic/tests"
	. "github.com/smartystreets/goconvey/convey"
	"github.com/syndtr/goleveldb/leveldb/errors"
)

func TestScrobbler(t *testing.T) {

	Init(t, false)

	mfRepo := persistence.CreateMockMediaFileRepo()
	npRepo := engine.CreateMockNowPlayingRepo()
	itCtrl := &mockItunesControl{}

	scrobbler := engine.NewScrobbler(itCtrl, mfRepo, npRepo)

	Convey("Given a DB with one song", t, func() {
		mfRepo.SetData(`[{"Id":"2","Title":"Hands Of Time"}]`, 1)

		Convey("When I scrobble an existing song", func() {
			now := time.Now()
			mf, err := scrobbler.Register("2", now)

			Convey("Then I get the scrobbled song back", func() {
				So(err, ShouldBeNil)
				So(mf.Title, ShouldEqual, "Hands Of Time")
			})

			Convey("And iTunes is notified", func() {
				So(itCtrl.played, ShouldContainKey, "2")
				So(itCtrl.played["2"].Equal(now), ShouldBeTrue)
			})

		})

		Convey("When the ID is not in the DB", func() {
			_, err := scrobbler.Register("3", time.Now())

			Convey("Then I receive an error", func() {
				So(err, ShouldNotBeNil)
			})

			Convey("And iTunes is not notified", func() {
				So(itCtrl.played, ShouldNotContainKey, "3")
			})
		})

		Convey("When I inform the song that is now playing", func() {
			mf, err := scrobbler.NowPlaying("2", "deluan", "DSub")

			Convey("Then I get the song for that id back", func() {
				So(err, ShouldBeNil)
				So(mf.Title, ShouldEqual, "Hands Of Time")
			})

			Convey("And it saves the song as the one current playing", func() {
				info := npRepo.Current()
				So(info.TrackId, ShouldEqual, "2")
				So(info.Start, ShouldHappenBefore, time.Now())
				So(info.Username, ShouldEqual, "deluan")
				So(info.PlayerName, ShouldEqual, "DSub")
			})

			Convey("And iTunes is not notified", func() {
				So(itCtrl.played, ShouldNotContainKey, "2")
			})
		})

		Reset(func() {
			itCtrl.played = make(map[string]time.Time)
		})

	})

}

type mockItunesControl struct {
	itunesbridge.ItunesControl
	played map[string]time.Time
	error  bool
}

func (m *mockItunesControl) MarkAsPlayed(id string, playDate time.Time) error {
	if m.error {
		return errors.New("ID not found")
	}
	if m.played == nil {
		m.played = make(map[string]time.Time)
	}
	m.played[id] = playDate
	return nil
}
