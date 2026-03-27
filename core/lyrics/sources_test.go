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

		It("should return Enhanced LRC lyrics with word-level cues from a file", func() {
			mf := model.MediaFile{Path: "tests/fixtures/test-enhanced.mp3"}
			lyrics, err := fromExternalFile(ctx, &mf, ".lrc")

			Expect(err).To(BeNil())
			Expect(lyrics).To(HaveLen(1))
			Expect(lyrics[0].DisplayArtist).To(Equal("Test Artist"))
			Expect(lyrics[0].DisplayTitle).To(Equal("Enhanced Test"))
			Expect(lyrics[0].Lang).To(Equal("eng"))
			Expect(lyrics[0].Synced).To(BeTrue())
			Expect(lyrics[0].Line).To(HaveLen(3))

			// Line 1: has inline markers → Cue array populated
			Expect(lyrics[0].Line[0].Start).To(Equal(gg.P(int64(1000))))
			Expect(lyrics[0].Line[0].End).To(Equal(gg.P(int64(3000))))
			Expect(lyrics[0].Line[0].Value).To(Equal("Some lyrics here"))
			Expect(lyrics[0].Line[0].Cue).To(HaveLen(3))
			Expect(*lyrics[0].Line[0].Cue[0].Start).To(Equal(int64(1000)))
			Expect(lyrics[0].Line[0].Cue[0].Value).To(Equal("Some "))
			Expect(lyrics[0].Line[0].Cue[0].End).To(Equal(gg.P(int64(1500))))
			Expect(*lyrics[0].Line[0].Cue[1].Start).To(Equal(int64(1500)))
			Expect(lyrics[0].Line[0].Cue[1].Value).To(Equal("lyrics "))
			Expect(lyrics[0].Line[0].Cue[1].End).To(Equal(gg.P(int64(2000))))
			Expect(*lyrics[0].Line[0].Cue[2].Start).To(Equal(int64(2000)))
			Expect(lyrics[0].Line[0].Cue[2].Value).To(Equal("here"))
			Expect(lyrics[0].Line[0].Cue[2].End).To(Equal(gg.P(int64(3000))))

			// Line 2: has inline markers
			Expect(lyrics[0].Line[1].Start).To(Equal(gg.P(int64(3000))))
			Expect(lyrics[0].Line[1].End).To(Equal(gg.P(int64(5000))))
			Expect(lyrics[0].Line[1].Value).To(Equal("More words"))
			Expect(lyrics[0].Line[1].Cue).To(HaveLen(2))
			Expect(lyrics[0].Line[1].Cue[0].End).To(Equal(gg.P(int64(3500))))
			Expect(lyrics[0].Line[1].Cue[1].End).To(Equal(gg.P(int64(5000))))

			// Line 3: plain line, no cues
			Expect(lyrics[0].Line[2].Start).To(Equal(gg.P(int64(5000))))
			Expect(lyrics[0].Line[2].Value).To(Equal("Plain line without inline markers"))
			Expect(lyrics[0].Line[2].Cue).To(BeNil())
		})

		It("should return Enhanced LRC lyrics from an ELRC file", func() {
			mf := model.MediaFile{Path: "tests/fixtures/test.mp3"}
			lyrics, err := fromExternalFile(ctx, &mf, ".elrc")

			Expect(err).To(BeNil())
			Expect(lyrics).To(HaveLen(1))
			Expect(lyrics[0].DisplayArtist).To(Equal("ELRC Artist"))
			Expect(lyrics[0].DisplayTitle).To(Equal("ELRC Song"))
			Expect(lyrics[0].Lang).To(Equal("eng"))
			Expect(lyrics[0].Synced).To(BeTrue())
			Expect(lyrics[0].Line).To(HaveLen(2))

			Expect(lyrics[0].Line[0].Start).To(Equal(gg.P(int64(1000))))
			Expect(lyrics[0].Line[0].End).To(Equal(gg.P(int64(3000))))
			Expect(lyrics[0].Line[0].Value).To(Equal("Lead words"))
			Expect(lyrics[0].Line[0].Cue).To(HaveLen(2))
			Expect(*lyrics[0].Line[0].Cue[0].Start).To(Equal(int64(1000)))
			Expect(lyrics[0].Line[0].Cue[0].Value).To(Equal("Lead "))
			Expect(lyrics[0].Line[0].Cue[0].End).To(Equal(gg.P(int64(1500))))
			Expect(*lyrics[0].Line[0].Cue[1].Start).To(Equal(int64(1500)))
			Expect(lyrics[0].Line[0].Cue[1].Value).To(Equal("words"))
			Expect(lyrics[0].Line[0].Cue[1].End).To(Equal(gg.P(int64(3000))))

			Expect(lyrics[0].Line[1].Start).To(Equal(gg.P(int64(3000))))
			Expect(lyrics[0].Line[1].Value).To(Equal("Fallback line"))
			Expect(lyrics[0].Line[1].Cue).To(BeNil())
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

		It("should return synchronized lyrics from an SRT file", func() {
			mf := model.MediaFile{Path: "tests/fixtures/test.mp3"}
			lyrics, err := fromExternalFile(ctx, &mf, ".srt")

			Expect(err).To(BeNil())
			Expect(lyrics).To(Equal(model.LyricList{
				model.Lyrics{
					Lang: "xxx",
					Line: []model.Line{
						{
							Start: gg.P(int64(18800)),
							End:   gg.P(int64(22800)),
							Value: "We're from subtitles",
						},
						{
							Start: gg.P(int64(22801)),
							End:   gg.P(int64(26000)),
							Value: "Another subtitle line",
						},
					},
					Synced: true,
				},
			}))
		})

		It("should return synchronized multilingual lyrics from a TTML file", func() {
			mf := model.MediaFile{Path: "tests/fixtures/test.mp3"}
			lyrics, err := fromExternalFile(ctx, &mf, ".ttml")

			Expect(err).To(BeNil())
			Expect(lyrics).To(Equal(model.LyricList{
				{
					Kind: "main",
					Lang: "eng",
					Line: []model.Line{
						{
							Start: gg.P(int64(18800)),
							Value: "We're no strangers to love",
						},
						{
							Start: gg.P(int64(22800)),
							Value: "You know the rules and so do I",
						},
					},
					Synced: true,
				},
				{
					Kind: "main",
					Lang: "por",
					Line: []model.Line{
						{
							Start: gg.P(int64(18800)),
							Value: "Nao somos estranhos ao amor",
						},
					},
					Synced: true,
				},
			}))
		})

		It("should handle LRC files with UTF-8 BOM marker (issue #4631)", func() {
			// The function looks for <basePath-without-ext><suffix>, so we need to pass
			// a MediaFile with .mp3 path and look for .lrc suffix
			mf := model.MediaFile{Path: "tests/fixtures/bom-test.mp3"}
			lyrics, err := fromExternalFile(ctx, &mf, ".lrc")

			Expect(err).To(BeNil())
			Expect(lyrics).ToNot(BeNil())
			Expect(lyrics).To(HaveLen(1))

			// The critical assertion: even with BOM, synced should be true
			Expect(lyrics[0].Synced).To(BeTrue(), "Lyrics with BOM marker should be recognized as synced")
			Expect(lyrics[0].Line).To(HaveLen(1))
			Expect(lyrics[0].Line[0].Start).To(Equal(gg.P(int64(0))))
			Expect(lyrics[0].Line[0].Value).To(ContainSubstring("作曲"))
		})

		It("should handle UTF-16 LE encoded LRC files", func() {
			mf := model.MediaFile{Path: "tests/fixtures/bom-utf16-test.mp3"}
			lyrics, err := fromExternalFile(ctx, &mf, ".lrc")

			Expect(err).To(BeNil())
			Expect(lyrics).ToNot(BeNil())
			Expect(lyrics).To(HaveLen(1))

			// UTF-16 should be properly converted to UTF-8
			Expect(lyrics[0].Synced).To(BeTrue(), "UTF-16 encoded lyrics should be recognized as synced")
			Expect(lyrics[0].Line).To(HaveLen(2))
			Expect(lyrics[0].Line[0].Start).To(Equal(gg.P(int64(18800))))
			Expect(lyrics[0].Line[0].Value).To(Equal("We're no strangers to love"))
			Expect(lyrics[0].Line[1].Start).To(Equal(gg.P(int64(22801))))
			Expect(lyrics[0].Line[1].Value).To(Equal("You know the rules and so do I"))
		})

		It("should handle TTML files with UTF-8 BOM marker", func() {
			mf := model.MediaFile{Path: "tests/fixtures/bom-test.mp3"}
			lyrics, err := fromExternalFile(ctx, &mf, ".ttml")

			Expect(err).To(BeNil())
			Expect(lyrics).To(HaveLen(1))
			Expect(lyrics[0].Kind).To(Equal("main"))
			Expect(lyrics[0].Synced).To(BeTrue())
			Expect(lyrics[0].Line).To(HaveLen(1))
			Expect(lyrics[0].Line[0].Start).To(Equal(gg.P(int64(0))))
			Expect(lyrics[0].Line[0].Value).To(Equal("BOM test line"))
		})

		It("should handle UTF-16 LE encoded TTML files", func() {
			mf := model.MediaFile{Path: "tests/fixtures/bom-utf16-test.mp3"}
			lyrics, err := fromExternalFile(ctx, &mf, ".ttml")

			Expect(err).To(BeNil())
			Expect(lyrics).To(HaveLen(1))
			Expect(lyrics[0].Kind).To(Equal("main"))
			Expect(lyrics[0].Synced).To(BeTrue())
			Expect(lyrics[0].Line).To(HaveLen(2))
			Expect(lyrics[0].Line[0].Start).To(Equal(gg.P(int64(18800))))
			Expect(lyrics[0].Line[0].Value).To(Equal("UTF16 line one"))
			Expect(lyrics[0].Line[1].Start).To(Equal(gg.P(int64(22801))))
			Expect(lyrics[0].Line[1].Value).To(Equal("UTF16 line two"))
		})
	})
})
