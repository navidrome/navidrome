package lyrics

import (
	"context"
	"encoding/json"
	"path/filepath"

	"github.com/navidrome/navidrome/model"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("sources", func() {
	var ctx context.Context

	BeforeEach(func() {
		ctx = GinkgoT().Context()
	})

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

			syncedList, _ := model.ParseLyrics(ctx, ".lrc", "eng", []byte(syncedLyrics))
			unsyncedList, _ := model.ParseLyrics(ctx, ".lrc", "xxx", []byte(unsyncedLyrics))
			synced, _ := syncedList.Main()
			unsynced, _ := unsyncedList.Main()

			expectedList := model.LyricList{synced, unsynced}
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
		var fixturesDir string

		BeforeEach(func() {
			// tests.Init sets CWD to the repo root, so "tests/fixtures" resolves correctly.
			abs, err := filepath.Abs("tests/fixtures")
			Expect(err).ToNot(HaveOccurred())
			fixturesDir = abs
		})

		mf := func(name string) *model.MediaFile {
			return &model.MediaFile{LibraryPath: fixturesDir, Path: name}
		}

		It("should return nil for lyrics that don't exist", func() {
			lyrics, err := fromExternalFile(ctx, mf("01 Invisible (RED) Edit Version.mp3"), ".lrc")

			Expect(err).To(BeNil())
			Expect(lyrics).To(HaveLen(0))
		})

		// fromExternalFile delegates format parsing to model.ParseLyrics; the
		// per-format parser output is covered exhaustively in the model package.
		// Here we only verify each suffix is read from the library FS and routed.
		DescribeTable("should read the sidecar file and route its suffix to a parser",
			func(name, suffix string, expectSynced bool) {
				lyrics, err := fromExternalFile(ctx, mf(name), suffix)

				Expect(err).To(BeNil())
				Expect(lyrics).ToNot(BeEmpty())
				Expect(lyrics[0].Line).ToNot(BeEmpty())
				Expect(lyrics[0].Synced).To(Equal(expectSynced))
			},
			Entry(".lrc synced", "test.mp3", ".lrc", true),
			Entry(".elrc enhanced", "test.mp3", ".elrc", true),
			Entry(".txt plain", "test.mp3", ".txt", false),
			Entry(".srt subtitles", "test.mp3", ".srt", true),
			Entry(".ttml multilingual", "test.mp3", ".ttml", true),
			Entry(".yaml lyricsfile", "test.mp3", ".yaml", true),
		)

		It("should handle LRC files with UTF-8 BOM marker (issue #4631)", func() {
			lyrics, err := fromExternalFile(ctx, mf("bom-test.mp3"), ".lrc")

			Expect(err).To(BeNil())
			Expect(lyrics).To(HaveLen(1))
			Expect(lyrics[0].Synced).To(BeTrue(), "Lyrics with BOM marker should be recognized as synced")
			Expect(lyrics[0].Line).To(HaveLen(1))
			Expect(lyrics[0].Line[0].Start).To(Equal(new(int64(0))))
			Expect(lyrics[0].Line[0].Value).To(ContainSubstring("作曲"))
		})

		It("should handle UTF-16 LE encoded LRC files", func() {
			lyrics, err := fromExternalFile(ctx, mf("bom-utf16-test.mp3"), ".lrc")

			Expect(err).To(BeNil())
			Expect(lyrics).To(HaveLen(1))
			Expect(lyrics[0].Synced).To(BeTrue(), "UTF-16 encoded lyrics should be recognized as synced")
			Expect(lyrics[0].Line).To(HaveLen(2))
			Expect(lyrics[0].Line[0].Start).To(Equal(new(int64(18800))))
			Expect(lyrics[0].Line[0].Value).To(Equal("We're no strangers to love"))
			Expect(lyrics[0].Line[1].Start).To(Equal(new(int64(22801))))
			Expect(lyrics[0].Line[1].Value).To(Equal("You know the rules and so do I"))
		})

		It("should handle TTML files with UTF-8 BOM marker", func() {
			lyrics, err := fromExternalFile(ctx, mf("bom-test.mp3"), ".ttml")

			Expect(err).To(BeNil())
			Expect(lyrics).To(HaveLen(1))
			Expect(lyrics[0].Kind).To(Equal("main"))
			Expect(lyrics[0].Synced).To(BeTrue())
			Expect(lyrics[0].Line).To(HaveLen(1))
			Expect(lyrics[0].Line[0].Start).To(Equal(new(int64(0))))
			Expect(lyrics[0].Line[0].Value).To(Equal("BOM test line"))
		})

		It("should handle UTF-16 BE encoded TTML files", func() {
			lyrics, err := fromExternalFile(ctx, mf("bom-utf16-test.mp3"), ".ttml")

			Expect(err).To(BeNil())
			Expect(lyrics).To(HaveLen(1))
			Expect(lyrics[0].Kind).To(Equal("main"))
			Expect(lyrics[0].Synced).To(BeTrue())
			Expect(lyrics[0].Line).To(HaveLen(2))
			Expect(lyrics[0].Line[0].Start).To(Equal(new(int64(18800))))
			Expect(lyrics[0].Line[0].Value).To(Equal("UTF16 line one"))
			Expect(lyrics[0].Line[1].Start).To(Equal(new(int64(22801))))
			Expect(lyrics[0].Line[1].Value).To(Equal("UTF16 line two"))
		})
	})
})
