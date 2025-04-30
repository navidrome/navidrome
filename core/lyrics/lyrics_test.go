package lyrics_test

import (
	"context"
	"encoding/json"
	"os"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/conf/configtest"
	"github.com/navidrome/navidrome/core/lyrics"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/utils"
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

	Context("Errors", func() {
		var RegularUserContext = XContext
		var isRegularUser = os.Getuid() != 0
		if isRegularUser {
			RegularUserContext = Context
		}

		RegularUserContext("run without root permissions", func() {
			var accessForbiddenFile string

			BeforeEach(func() {
				accessForbiddenFile = utils.TempFileName("access_forbidden-", ".mp3")

				f, err := os.OpenFile(accessForbiddenFile, os.O_WRONLY|os.O_CREATE, 0222)
				Expect(err).ToNot(HaveOccurred())

				mf.Path = accessForbiddenFile

				DeferCleanup(func() {
					Expect(f.Close()).To(Succeed())
					Expect(os.Remove(accessForbiddenFile)).To(Succeed())
				})
			})

			It("should fallback to embedded if an error happens when parsing file", func() {
				conf.Server.LyricsPriority = ".mp3,embedded"

				list, err := lyrics.GetLyrics(ctx, &mf)
				Expect(err).To(BeNil())
				Expect(list).To(Equal(embeddedLyrics))
			})

			It("should return nothing if error happens when trying to parse file", func() {
				conf.Server.LyricsPriority = ".mp3"

				list, err := lyrics.GetLyrics(ctx, &mf)
				Expect(err).To(BeNil())
				Expect(list).To(BeEmpty())
			})
		})
	})
})
