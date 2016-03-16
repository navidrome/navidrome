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
	itCtrl := &mockItunesControl{}

	scrobbler := engine.NewScrobbler(itCtrl, mfRepo)

	Convey("Given a DB with one song", t, func() {
		mfRepo.SetData(`[{"Id":"2","Title":"Hands Of Time"}]`, 1)

		Convey("When I scrobble an existing song", func() {
			now := time.Now()
			mf, err := scrobbler.Register("2", now, true)

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
			_, err := scrobbler.Register("3", time.Now(), true)

			Convey("Then I receive an error", func() {
				So(err, ShouldNotBeNil)
			})

			Convey("And iTunes is not notified", func() {
				So(itCtrl.played, ShouldNotContainKey, "3")
			})
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
