package lyrics

import (
	"context"
	"encoding/json"

	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/utils/gg"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("sources", func() {
	ctx := context.Background()

	Describe("fromEmbedded", func() {
		It("should return nothing for a media file with no lyrics", func() {
			mf := model.MediaFile{}
			lyrics, err := fromEmbedded(ctx, &mf)

			Expect(err).To(BeNil())
			Expect(lyrics).To(HaveLen(0))
		})

		It("should return lyrics for a media file with well-formatted lyrics", func() {
			const syncedLyrics = "[00:18.80]We're no strangers to love\n[00:22.801]You know the rules and so do I"
			const unsyncedLyrics = "We're no strangers to love\nYou know the rules and so do I"

			synced, _ := model.ToLyrics("eng", syncedLyrics)
			unsynced, _ := model.ToLyrics("xxx", unsyncedLyrics)

			expectedList := model.LyricList{*synced, *unsynced}
			lyricsJson, err := json.Marshal(expectedList)

			Expect(err).ToNot(HaveOccurred())

			mf := model.MediaFile{
				Lyrics: string(lyricsJson),
			}

			lyrics, err := fromEmbedded(ctx, &mf)
			Expect(err).To(BeNil())
			Expect(lyrics).ToNot(BeNil())
			Expect(lyrics).To(Equal(expectedList))
		})

		It("should return an error if somehow the JSON is bad", func() {
			mf := model.MediaFile{Lyrics: "["}
			lyrics, err := fromEmbedded(ctx, &mf)

			Expect(lyrics).To(HaveLen(0))
			Expect(err).ToNot(BeNil())
		})
	})

	Describe("fromExternalFile", func() {
		It("should return nil for lyrics that don't exist", func() {
			mf := model.MediaFile{Path: "tests/fixtures/01 Invisible (RED) Edit Version.mp3"}
			lyrics, err := fromExternalFile(ctx, &mf, ".lrc")

			Expect(err).To(BeNil())
			Expect(lyrics).To(HaveLen(0))
		})

		It("should return synchronized lyrics from a file", func() {
			mf := model.MediaFile{Path: "tests/fixtures/test.mp3"}
			lyrics, err := fromExternalFile(ctx, &mf, ".lrc")

			Expect(err).To(BeNil())
			Expect(lyrics).To(Equal(model.LyricList{
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
			}))
		})

		It("should return unsynchronized lyrics from a file", func() {
			mf := model.MediaFile{Path: "tests/fixtures/test.mp3"}
			lyrics, err := fromExternalFile(ctx, &mf, ".txt")

			Expect(err).To(BeNil())
			Expect(lyrics).To(Equal(model.LyricList{
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
			}))
		})
	})
})
