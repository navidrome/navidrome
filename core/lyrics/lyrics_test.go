package lyrics_test

import (
	"context"
	"encoding/json"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/conf/configtest"
	"github.com/navidrome/navidrome/core/lyrics"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/utils/gg"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("sources", func() {
	var mf model.MediaFile
	var ctx context.Context

	const badLyrics = "This is a set of lyrics\nThat is not good"
	unsynced, _ := model.ToLyrics("xxx", badLyrics)
	embeddedLyrics := model.LyricList{*unsynced}

	syncedLyrics := model.LyricList{
		model.Lyrics{
			DisplayArtist: "Rick Astley",
			DisplayTitle:  "That one song",
			Lang:          "eng",
			Line: []model.Line{
				{
					Start: gg.P(int64(18800)),
					Value: "We're no strangers to love",
				},
				{
					Start: gg.P(int64(22801)),
					Value: "You know the rules and so do I",
				},
			},
			Offset: gg.P(int64(-100)),
			Synced: true,
		},
	}

	unsyncedLyrics := model.LyricList{
		model.Lyrics{
			Lang: "xxx",
			Line: []model.Line{
				{
					Value: "We're no strangers to love",
				},
				{
					Value: "You know the rules and so do I",
				},
			},
			Synced: false,
		},
	}

	BeforeEach(func() {
		DeferCleanup(configtest.SetupConfig())

		lyricsJson, _ := json.Marshal(embeddedLyrics)

		mf = model.MediaFile{
			Lyrics: string(lyricsJson),
			Path:   "tests/fixtures/test.mp3",
		}
		ctx = context.Background()
	})

	DescribeTable("Lyrics Priority", func(priority string, expected model.LyricList) {
		conf.Server.LyricsPriority = priority
		list, err := lyrics.GetLyrics(ctx, &mf)
		Expect(err).To(BeNil())
		Expect(list).To(Equal(expected))
	},
		Entry("embedded > lrc > txt", "embedded,.lrc,.txt", embeddedLyrics),
		Entry("lrc > embedded > txt", ".lrc,embedded,.txt", syncedLyrics),
		Entry("txt > lrc > embedded", ".txt,.lrc,embedded", unsyncedLyrics))
})
