package lyrics

import (
	"context"
	"encoding/json"
	"path/filepath"
	"testing/fstest"
	"time"

	"github.com/navidrome/navidrome/core/storage/storagetest"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
	"golang.org/x/text/unicode/norm"
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

			Expect(err).ToNot(HaveOccurred())
			Expect(lyrics).To(BeEmpty())
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
			Expect(err).ToNot(HaveOccurred())
			Expect(lyrics).ToNot(BeNil())
			Expect(lyrics).To(Equal(expectedList))
		})

		It("should return an error if somehow the JSON is bad", func() {
			mf := model.MediaFile{Lyrics: "["}
			lyrics, err := fromEmbedded(ctx, &mf)

			Expect(lyrics).To(BeEmpty())
			Expect(err).To(HaveOccurred())
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

			Expect(err).ToNot(HaveOccurred())
			Expect(lyrics).To(BeEmpty())
		})

		// fromExternalFile delegates format parsing to model.ParseLyrics; the
		// per-format parser output is covered exhaustively in the model package.
		// Here we only verify each suffix is read from the library FS and routed.
		DescribeTable("should read the sidecar file and route its suffix to a parser",
			func(name, suffix string, expectSynced bool) {
				lyrics, err := fromExternalFile(ctx, mf(name), suffix)

				Expect(err).ToNot(HaveOccurred())
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

			Expect(err).ToNot(HaveOccurred())
			Expect(lyrics).To(HaveLen(1))
			Expect(lyrics[0].Synced).To(BeTrue(), "Lyrics with BOM marker should be recognized as synced")
			Expect(lyrics[0].Line).To(HaveLen(1))
			Expect(lyrics[0].Line[0].Start).To(Equal(new(int64(0))))
			Expect(lyrics[0].Line[0].Value).To(ContainSubstring("作曲"))
		})

		It("should handle UTF-16 LE encoded LRC files", func() {
			lyrics, err := fromExternalFile(ctx, mf("bom-utf16-test.mp3"), ".lrc")

			Expect(err).ToNot(HaveOccurred())
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

			Expect(err).ToNot(HaveOccurred())
			Expect(lyrics).To(HaveLen(1))
			Expect(lyrics[0].Kind).To(Equal("main"))
			Expect(lyrics[0].Synced).To(BeTrue())
			Expect(lyrics[0].Line).To(HaveLen(1))
			Expect(lyrics[0].Line[0].Start).To(Equal(new(int64(0))))
			Expect(lyrics[0].Line[0].Value).To(Equal("BOM test line"))
		})

		It("should handle UTF-16 BE encoded TTML files", func() {
			lyrics, err := fromExternalFile(ctx, mf("bom-utf16-test.mp3"), ".ttml")

			Expect(err).ToNot(HaveOccurred())
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

	// The sidecar's file name can be in another Unicode form than the track's. See openSidecar.
	Describe("fromExternalFile with mismatched Unicode normalization", func() {
		const scheme = "fake-lyrics-norm"
		// "ガ" (katakana KA plus a dakuten) has distinct NFC and NFD forms.
		const base = "03. 鬱P feat. 初音ミク - ガ"
		const dir = "ビートルズ/Beyoncé/"

		var fsys *storagetest.FakeFS

		BeforeEach(func() {
			fsys = &storagetest.FakeFS{}
			storagetest.Register(scheme, fsys)
		})

		// lyricsFor stores the track and its sidecar under dir (kept byte-for-byte,
		// as the scanner reads it from disk) with the file names in the given forms.
		lyricsFor := func(dir string, trackForm, sidecarForm norm.Form) (model.LyricList, error) {
			trackBase := dir + trackForm.String(base)
			sidecarBase := dir + sidecarForm.String(base)
			Expect(trackBase).ToNot(Equal(sidecarBase))

			// A non-zero ModTime is required so the FakeFS directory-timestamp
			// bookkeeping settles instead of looping.
			modTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
			fsys.SetFiles(fstest.MapFS{
				trackBase + ".flac":  &fstest.MapFile{Data: []byte("audio"), ModTime: modTime},
				sidecarBase + ".lrc": &fstest.MapFile{Data: []byte("[00:18.80]We're no strangers to love"), ModTime: modTime},
			})

			mf := &model.MediaFile{LibraryPath: scheme + ":///music", Path: trackBase + ".flac"}
			return fromExternalFile(ctx, mf, ".lrc")
		}

		// Only the sidecar's file name can differ in form: the folders come from the
		// scanner, so they are correct as stored and must not be re-normalized.
		DescribeTable("finds a sidecar stored in the other normalization form",
			func(dir string, trackForm, sidecarForm norm.Form) {
				lyrics, err := lyricsFor(dir, trackForm, sidecarForm)

				Expect(err).ToNot(HaveOccurred())
				Expect(lyrics).To(HaveLen(1))
				Expect(lyrics[0].Synced).To(BeTrue())
			},
			Entry("no folder, NFC track, NFD sidecar", "", norm.NFC, norm.NFD),
			Entry("no folder, NFD track, NFC sidecar", "", norm.NFD, norm.NFC),
			Entry("NFC folder, NFC track, NFD sidecar", norm.NFC.String(dir), norm.NFC, norm.NFD),
			Entry("NFC folder, NFD track, NFC sidecar", norm.NFC.String(dir), norm.NFD, norm.NFC),
			Entry("NFD folder, NFD track, NFC sidecar", norm.NFD.String(dir), norm.NFD, norm.NFC),
			Entry("NFD folder, NFC track, NFD sidecar", norm.NFD.String(dir), norm.NFC, norm.NFD),
		)

		It("logs the sidecar path that is really on disk", func() {
			l, hook := test.NewNullLogger()
			prevLevel := log.CurrentLevel()
			// Swap the logger before raising the level, so Trace entries reach the hook.
			prevLogger := log.SetDefaultLogger(l)
			log.SetLevel(log.LevelTrace)
			DeferCleanup(func() {
				log.SetDefaultLogger(prevLogger)
				log.SetLevel(prevLevel)
			})

			_, err := lyricsFor("", norm.NFC, norm.NFD)

			Expect(err).ToNot(HaveOccurred())
			entry := hook.LastEntry()
			Expect(entry).ToNot(BeNil())
			Expect(entry.Level).To(Equal(logrus.TraceLevel))
			Expect(entry.Message).To(Equal("retrieved lyrics from external file"))
			Expect(entry.Data).To(HaveKeyWithValue("file", norm.NFD.String(base)+".lrc"))
		})
	})
})
