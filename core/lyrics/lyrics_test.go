package lyrics_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing/fstest"
	"time"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/core/lyrics"
	"github.com/navidrome/navidrome/core/storage/storagetest"
	"github.com/navidrome/navidrome/db"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/persistence"
	"github.com/navidrome/navidrome/persistence/persistencetest"
	"github.com/navidrome/navidrome/tests"
	"github.com/navidrome/navidrome/utils"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"golang.org/x/text/unicode/norm"
)

var _ = Describe("Lyrics", func() {
	var mf model.MediaFile
	var ctx context.Context
	var ds model.DataStore

	embeddedLyrics := model.LyricList{
		model.Lyrics{
			Lang: "xxx",
			Line: []model.Line{
				{Value: "This is a set of lyrics"},
				{Value: "That is not good"},
			},
		},
	}

	syncedLyrics := model.LyricList{
		model.Lyrics{
			DisplayArtist: "Rick Astley",
			DisplayTitle:  "That one song",
			Lang:          "eng",
			Line: []model.Line{
				{
					Start: new(int64(18800)),
					Value: "We're no strangers to love",
				},
				{
					Start: new(int64(22801)),
					Value: "You know the rules and so do I",
				},
			},
			Offset: new(int64(-100)),
			Synced: true,
		},
	}

	elrcLyrics := model.LyricList{
		model.Lyrics{
			DisplayArtist: "ELRC Artist",
			DisplayTitle:  "ELRC Song",
			Lang:          "eng",
			Line: []model.Line{
				{
					Start: new(int64(1000)),
					End:   new(int64(3000)),
					Value: "Lead words",
					Cue: []model.Cue{
						{
							Start:     new(int64(1000)),
							End:       new(int64(1500)),
							Value:     "Lead ",
							ByteStart: 0,
							ByteEnd:   4,
						},
						{
							Start:     new(int64(1500)),
							End:       new(int64(3000)),
							Value:     "words",
							ByteStart: 5,
							ByteEnd:   9,
						},
					},
				},
				{
					Start: new(int64(3000)),
					Value: "Fallback line",
				},
			},
			Synced: true,
		},
	}

	ttmlLyrics := model.LyricList{
		model.Lyrics{
			Kind: "main",
			Lang: "eng",
			Line: []model.Line{
				{
					Start: new(int64(18800)),
					Value: "We're no strangers to love",
				},
				{
					Start: new(int64(22800)),
					Value: "You know the rules and so do I",
				},
			},
			Synced: true,
		},
		model.Lyrics{
			Kind: "main",
			Lang: "por",
			Line: []model.Line{
				{
					Start: new(int64(18800)),
					Value: "Nao somos estranhos ao amor",
				},
			},
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

	srtLyrics := model.LyricList{
		model.Lyrics{
			Lang: "xxx",
			Line: []model.Line{
				{
					Start: new(int64(18800)),
					End:   new(int64(22800)),
					Value: "We're from subtitles",
				},
				{
					Start: new(int64(22801)),
					End:   new(int64(26000)),
					Value: "Another subtitle line",
				},
			},
			Synced: true,
		},
	}

	BeforeEach(func() {

		lyricsJson, _ := json.Marshal(embeddedLyrics)

		mf = model.MediaFile{
			Lyrics: string(lyricsJson),
			Path:   "tests/fixtures/test.mp3",
		}
		ctx = GinkgoT().Context()
		ds = persistence.New(db.Db())
	})

	DescribeTable("Lyrics Priority", func(priority string, expected model.LyricList) {
		conf.Update(func(c *conf.Options) { c.LyricsPriority = priority })
		svc := lyrics.NewLyrics(ds, nil)
		list, err := svc.GetLyrics(ctx, &mf)
		Expect(err).ToNot(HaveOccurred())
		Expect(list).To(Equal(expected))
	},
		Entry("embedded > lrc > txt", "embedded,.lrc,.txt", embeddedLyrics),
		Entry("lrc > embedded > txt", ".lrc,embedded,.txt", syncedLyrics),
		Entry("elrc > lrc > embedded", ".elrc,.lrc,embedded", elrcLyrics),
		Entry("srt > txt > embedded", ".srt,.txt,embedded", srtLyrics),
		Entry("txt > lrc > embedded", ".txt,.lrc,embedded", unsyncedLyrics),
		Entry("ttml > elrc > lrc > srt > embedded", ".ttml,.elrc,.lrc,.srt,embedded", ttmlLyrics))

	It("resolves source priority across duplicate media files", func() {
		conf.Update(func(c *conf.Options) { c.LyricsPriority = ".ttml,embedded" })
		embeddedJSON, err := json.Marshal(embeddedLyrics)
		Expect(err).ToNot(HaveOccurred())

		useRepoRootAsLibrary(ctx, ds)
		putRickAstleySong(ctx, ds, "tests/fixtures/01 Invisible (RED) Edit Version.mp3", string(embeddedJSON))
		putRickAstleySong(ctx, ds, "tests/fixtures/test.mp3", "[]")
		svc := lyrics.NewLyrics(ds, nil)

		list, err := svc.GetLyricsByArtistTitle(ctx, "Rick Astley", "Never Gonna Give You Up")
		Expect(err).ToNot(HaveOccurred())
		Expect(list).To(Equal(ttmlLyrics))
	})

	It("lists sidecar suffixes lowercased, sorted and without duplicates", func() {
		conf.Update(func(c *conf.Options) { c.LyricsPriority = " .TXT , embedded,.lrc,my-plugin,.txt" })
		Expect(lyrics.FileSuffixes()).To(Equal([]string{".lrc", ".txt"}))
	})

	Context("folder lyrics files", func() {
		BeforeEach(func() {
			conf.Update(func(c *conf.Options) { c.LyricsPriority = ".lrc,.txt,embedded" })
		})
		withFolder := func(files []string) lyrics.Lyrics {
			GinkgoHelper()
			lib, err := ds.Library().Get(ctx, model.DefaultLibraryID)
			Expect(err).ToNot(HaveOccurred())
			folder := model.NewFolder(*lib, "tests/fixtures")
			folder.LyricsFiles = files
			Expect(ds.Folder().Put(ctx, folder)).To(Succeed())
			mf.FolderID = folder.ID
			return lyrics.NewLyrics(ds, nil)
		}

		It("reads only the lyrics files the scanner found", func() {
			list, err := withFolder([]string{"test.txt"}).GetLyrics(ctx, &mf)
			Expect(err).ToNot(HaveOccurred())
			Expect(list).To(Equal(unsyncedLyrics))
		})

		It("skips the filesystem when the folder has no lyrics files", func() {
			list, err := withFolder([]string{}).GetLyrics(ctx, &mf)
			Expect(err).ToNot(HaveOccurred())
			Expect(list).To(Equal(embeddedLyrics))
		})

		It("probes the filesystem when the folder was not scanned for lyrics files", func() {
			list, err := withFolder(nil).GetLyrics(ctx, &mf)
			Expect(err).ToNot(HaveOccurred())
			Expect(list).To(Equal(syncedLyrics))
		})
	})

	It("preserves configured sidecar suffix casing on case-sensitive filesystems", func() {
		dir, err := os.MkdirTemp("", "lyrics-case-*")
		Expect(err).ToNot(HaveOccurred())
		DeferCleanup(func() {
			Expect(os.RemoveAll(dir)).To(Succeed())
		})

		probe := filepath.Join(dir, "CASECHECK")
		Expect(os.WriteFile(probe, []byte("probe"), 0600)).To(Succeed())
		_, err = os.Stat(filepath.Join(dir, "casecheck"))
		if err == nil {
			Skip("filesystem is case-insensitive")
		}
		Expect(os.IsNotExist(err)).To(BeTrue())

		conf.Update(func(c *conf.Options) { c.LyricsPriority = ".LRC" })
		Expect(os.WriteFile(filepath.Join(dir, "song.LRC"), []byte("[00:01.00]Upper suffix"), 0600)).To(Succeed())

		svc := lyrics.NewLyrics(ds, nil)
		list, err := svc.GetLyrics(ctx, &model.MediaFile{
			LibraryPath: dir,
			Path:        "song.mp3",
		})

		Expect(err).ToNot(HaveOccurred())
		Expect(list).To(HaveLen(1))
		Expect(list[0].Line).To(Equal([]model.Line{
			{Start: new(int64(1000)), Value: "Upper suffix"},
		}))
	})

	It("returns a non-Lyricsfile YAML sidecar as plain text, shadowing lower-priority sources", func() {
		dir, err := os.MkdirTemp("", "lyrics-yaml-fallback-*")
		Expect(err).ToNot(HaveOccurred())
		DeferCleanup(func() {
			Expect(os.RemoveAll(dir)).To(Succeed())
		})

		Expect(os.WriteFile(filepath.Join(dir, "song.yaml"), []byte("title: not lyricsfile\n"), 0600)).To(Succeed())
		Expect(os.WriteFile(filepath.Join(dir, "song.lrc"), []byte("[00:01.00]Fallback line"), 0600)).To(Succeed())

		conf.Update(func(c *conf.Options) { c.LyricsPriority = ".yaml,.lrc" })
		svc := lyrics.NewLyrics(ds, nil)
		list, err := svc.GetLyrics(ctx, &model.MediaFile{
			LibraryPath: dir,
			Path:        "song.mp3",
		})

		// ParseLyrics falls back to plain text for any suffix when the content
		// doesn't match the structured format, so the .yaml hit is non-empty and
		// shadows the lower-priority .lrc entirely.
		Expect(err).ToNot(HaveOccurred())
		Expect(list).To(HaveLen(1))
		Expect(list[0].Synced).To(BeFalse())
		Expect(list[0].Line).To(Equal([]model.Line{
			{Value: "title: not lyricsfile"},
		}))
	})

	It("resolves an external sidecar whose Unicode normalization differs from the track path", func() {
		// An in-memory FS keeps the NFC/NFD mismatch byte-exact. The host filesystem
		// may normalize names on lookup and hide the bug. See openSidecar.
		const scheme = "fake-lyrics-e2e"
		fsys := &storagetest.FakeFS{}
		storagetest.Register(scheme, fsys)

		const base = "03. 鬱P feat. 初音ミク - ガ"
		modTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
		fsys.SetFiles(fstest.MapFS{
			norm.NFC.String(base) + ".flac": &fstest.MapFile{Data: []byte("audio"), ModTime: modTime},
			norm.NFD.String(base) + ".lrc":  &fstest.MapFile{Data: []byte("[00:18.80]We're no strangers to love"), ModTime: modTime},
		})

		// Default priority from the bug report: ".lrc,.txt,embedded".
		conf.Update(func(c *conf.Options) { c.LyricsPriority = ".lrc,.txt,embedded" })
		svc := lyrics.NewLyrics(ds, nil)
		list, err := svc.GetLyrics(ctx, &model.MediaFile{
			LibraryPath: scheme + ":///music",
			Path:        norm.NFC.String(base) + ".flac",
		})

		Expect(err).ToNot(HaveOccurred())
		Expect(list).To(HaveLen(1))
		Expect(list[0].Synced).To(BeTrue())
		Expect(list[0].Line).To(HaveLen(1))
		Expect(list[0].Line[0].Value).To(Equal("We're no strangers to love"))
	})

	Context("Errors", func() {
		var RegularUserContext = XContext
		var isRegularUser = os.Getuid() != 0
		if isRegularUser {
			RegularUserContext = Context
		}

		RegularUserContext("run without root permissions", func() {
			var accessForbiddenFile string

			BeforeEach(func() {
				tests.SkipOnWindows("uses Unix file permission bits")
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
				conf.Update(func(c *conf.Options) { c.LyricsPriority = ".mp3,embedded" })

				svc := lyrics.NewLyrics(ds, nil)
				list, err := svc.GetLyrics(ctx, &mf)
				Expect(err).ToNot(HaveOccurred())
				Expect(list).To(Equal(embeddedLyrics))
			})

			It("should return nothing if error happens when trying to parse file", func() {
				conf.Update(func(c *conf.Options) { c.LyricsPriority = ".mp3" })

				svc := lyrics.NewLyrics(ds, nil)
				list, err := svc.GetLyrics(ctx, &mf)
				Expect(err).ToNot(HaveOccurred())
				Expect(list).To(BeEmpty())
			})
		})
	})

	Context("plugin sources", func() {
		var mockLoader *mockPluginLoader

		BeforeEach(func() {
			mockLoader = &mockPluginLoader{}
		})

		It("should return lyrics from a plugin", func() {
			conf.Update(func(c *conf.Options) { c.LyricsPriority = "test-lyrics-plugin" })
			mockLoader.lyrics = unsyncedLyrics
			svc := lyrics.NewLyrics(ds, mockLoader)
			list, err := svc.GetLyrics(ctx, &mf)
			Expect(err).ToNot(HaveOccurred())
			Expect(list).To(Equal(unsyncedLyrics))
		})

		It("should try plugin after embedded returns nothing", func() {
			conf.Update(func(c *conf.Options) { c.LyricsPriority = "embedded,test-lyrics-plugin" })
			mf.Lyrics = "" // No embedded lyrics
			mockLoader.lyrics = unsyncedLyrics
			svc := lyrics.NewLyrics(ds, mockLoader)
			list, err := svc.GetLyrics(ctx, &mf)
			Expect(err).ToNot(HaveOccurred())
			Expect(list).To(Equal(unsyncedLyrics))
		})

		It("should skip plugin if embedded has lyrics", func() {
			conf.Update(func(c *conf.Options) { c.LyricsPriority = "embedded,test-lyrics-plugin" })
			mockLoader.lyrics = unsyncedLyrics
			svc := lyrics.NewLyrics(ds, mockLoader)
			list, err := svc.GetLyrics(ctx, &mf)
			Expect(err).ToNot(HaveOccurred())
			Expect(list).To(Equal(embeddedLyrics)) // embedded wins
		})

		It("should skip unknown plugin names gracefully", func() {
			conf.Update(func(c *conf.Options) { c.LyricsPriority = "nonexistent-plugin,embedded" })
			mockLoader.notFound = true
			svc := lyrics.NewLyrics(ds, mockLoader)
			list, err := svc.GetLyrics(ctx, &mf)
			Expect(err).ToNot(HaveOccurred())
			Expect(list).To(Equal(embeddedLyrics)) // falls through to embedded
		})

		It("should preserve plugin name case from config", func() {
			conf.Update(func(c *conf.Options) { c.LyricsPriority = "MyLyricsPlugin" })
			mockLoader.pluginName = "MyLyricsPlugin"
			mockLoader.lyrics = unsyncedLyrics
			svc := lyrics.NewLyrics(ds, mockLoader)
			list, err := svc.GetLyrics(ctx, &mf)
			Expect(err).ToNot(HaveOccurred())
			Expect(list).To(Equal(unsyncedLyrics))
		})

		It("should handle plugin error gracefully", func() {
			conf.Update(func(c *conf.Options) { c.LyricsPriority = "test-lyrics-plugin,embedded" })
			mockLoader.err = errors.New("plugin error")
			svc := lyrics.NewLyrics(ds, mockLoader)
			list, err := svc.GetLyrics(ctx, &mf)
			Expect(err).ToNot(HaveOccurred())
			Expect(list).To(Equal(embeddedLyrics)) // falls through to embedded
		})
	})

	Context("caching", func() {
		var mockLoader *mockPluginLoader
		var svc lyrics.Lyrics

		BeforeEach(func() {
			conf.Update(func(c *conf.Options) { c.LyricsPriority = "test-lyrics-plugin" })
			mockLoader = &mockPluginLoader{}
			svc = lyrics.NewLyrics(ds, mockLoader)
			mf.ID = "mf-1"
		})

		It("serves a repeated request for the same track from the cache", func() {
			mockLoader.lyrics = unsyncedLyrics
			_, err := svc.GetLyrics(ctx, &mf)
			Expect(err).ToNot(HaveOccurred())

			list, err := svc.GetLyrics(ctx, &mf)
			Expect(err).ToNot(HaveOccurred())
			Expect(list).To(Equal(unsyncedLyrics))
			Expect(mockLoader.calls).To(Equal(1))
		})

		It("caches a miss, so a track without lyrics does not reach the plugin again", func() {
			for range 2 {
				list, err := svc.GetLyrics(ctx, &mf)
				Expect(err).ToNot(HaveOccurred())
				Expect(list).To(BeEmpty())
			}
			Expect(mockLoader.calls).To(Equal(1))
		})

		It("caches a plugin error as a miss, since some plugins report no lyrics as an error", func() {
			mockLoader.err = errors.New("no lyrics found from any provider")
			for range 2 {
				list, err := svc.GetLyrics(ctx, &mf)
				Expect(err).ToNot(HaveOccurred())
				Expect(list).To(BeEmpty())
			}
			Expect(mockLoader.calls).To(Equal(1))
		})

		It("looks up each track separately", func() {
			other := mf
			other.ID = "mf-2"
			_, _ = svc.GetLyrics(ctx, &mf)
			_, _ = svc.GetLyrics(ctx, &other)
			Expect(mockLoader.calls).To(Equal(2))
		})

		It("finishes and caches the lookup when the caller's request is cancelled", func() {
			mockLoader.lyrics = unsyncedLyrics
			cancelled, cancel := context.WithCancel(ctx)
			cancel()

			list, err := svc.GetLyrics(cancelled, &mf)
			Expect(err).ToNot(HaveOccurred())
			Expect(list).To(Equal(unsyncedLyrics))

			list, err = svc.GetLyrics(ctx, &mf)
			Expect(err).ToNot(HaveOccurred())
			Expect(list).To(Equal(unsyncedLyrics))
			Expect(mockLoader.calls).To(Equal(1))
		})

		It("bounds the lookup with a deadline", func() {
			_, err := svc.GetLyrics(ctx, &mf)
			Expect(err).ToNot(HaveOccurred())
			Expect(mockLoader.hadDeadline).To(BeTrue())
		})
	})

	var _ = Describe("GetLyricsByArtistTitle", func() {
		var svc lyrics.Lyrics

		BeforeEach(func() {
			conf.Update(func(c *conf.Options) { c.LyricsPriority = "embedded" })
			svc = lyrics.NewLyrics(ds, nil)
		})

		It("bounds the query to a duplicate window", func() {
			conf.Update(func(c *conf.Options) { c.LyricsPriority = ".lrc" })
			useRepoRootAsLibrary(ctx, ds)
			var decoys []string
			for i := range 10 {
				decoys = append(decoys, putRickAstleySong(ctx, ds, fmt.Sprintf("missing/%02d.mp3", i), ""))
			}
			withSidecar := putRickAstleySong(ctx, ds, "tests/fixtures/test.mp3", "")
			_, err := db.Db().Exec("UPDATE media_file SET updated_at = ? WHERE id = ?", time.Now().Add(-time.Hour), withSidecar)
			Expect(err).ToNot(HaveOccurred())

			list, err := svc.GetLyricsByArtistTitle(ctx, "Rick Astley", "Never Gonna Give You Up")
			Expect(err).ToNot(HaveOccurred())
			Expect(list).To(BeNil())

			Expect(ds.MediaFile().Delete(ctx, decoys[0])).To(Succeed())
			list, err = svc.GetLyricsByArtistTitle(ctx, "Rick Astley", "Never Gonna Give You Up")
			Expect(err).ToNot(HaveOccurred())
			Expect(list).To(Equal(syncedLyrics))
		})

		It("returns nil when no media file matches", func() {
			lyricsJSON, err := json.Marshal(embeddedLyrics)
			Expect(err).ToNot(HaveOccurred())
			putRickAstleySong(ctx, ds, "song.mp3", string(lyricsJSON))

			list, err := svc.GetLyricsByArtistTitle(ctx, "Rick Astley", "No Song")
			Expect(err).ToNot(HaveOccurred())
			Expect(list).To(BeNil())

			list, err = svc.GetLyricsByArtistTitle(ctx, "Nobody", "Never Gonna Give You Up")
			Expect(err).ToNot(HaveOccurred())
			Expect(list).To(BeNil())
		})

		It("resolves lyrics from the matched media files", func() {
			embeddedList, err := model.ParseLyrics(ctx, ".lrc", "eng", []byte("Embedded lyrics line"))
			Expect(err).ToNot(HaveOccurred())
			embedded, _ := embeddedList.Main()
			embeddedJSON, err := json.Marshal(model.LyricList{embedded})
			Expect(err).ToNot(HaveOccurred())
			putRickAstleySong(ctx, ds, "song.mp3", string(embeddedJSON))

			list, err := svc.GetLyricsByArtistTitle(ctx, "Rick Astley", "Never Gonna Give You Up")
			Expect(err).ToNot(HaveOccurred())
			Expect(list).To(HaveLen(1))
			Expect(list[0].Line[0].Value).To(Equal("Embedded lyrics line"))
		})
	})
})

func useRepoRootAsLibrary(ctx context.Context, ds model.DataStore) {
	GinkgoHelper()
	root, err := os.Getwd()
	Expect(err).ToNot(HaveOccurred())
	persistencetest.SetLibraries(ctx, ds, root)
}

// putRickAstleySong stores "Never Gonna Give You Up" by Rick Astley at path and returns its ID.
func putRickAstleySong(ctx context.Context, ds model.DataStore, path, lyrics string) string {
	GinkgoHelper()
	artist := model.Artist{ID: "ar-rick", Name: "Rick Astley"}
	Expect(ds.Artist().Put(ctx, &artist)).To(Succeed())
	mf := &model.MediaFile{
		LibraryID:    model.DefaultLibraryID,
		Path:         path,
		Title:        "Never Gonna Give You Up",
		Lyrics:       lyrics,
		UpdatedAt:    time.Now(),
		Participants: model.Participants{model.RoleArtist: {{Artist: artist}}},
	}
	Expect(ds.MediaFile().Put(ctx, mf)).To(Succeed())
	return mf.ID
}

type mockPluginLoader struct {
	lyrics      model.LyricList
	err         error
	notFound    bool
	pluginName  string // expected plugin name (exact match, like real manager)
	calls       int
	hadDeadline bool
}

func (m *mockPluginLoader) PluginNames(_ string) []string {
	if m.notFound {
		return nil
	}
	return []string{"test-lyrics-plugin"}
}

func (m *mockPluginLoader) LoadLyricsProvider(name string) (lyrics.Provider, bool) {
	if m.notFound {
		return nil, false
	}
	// If pluginName is set, require exact match (like the real plugin manager)
	if m.pluginName != "" && name != m.pluginName {
		return nil, false
	}
	return &mockLyricsProvider{loader: m}, true
}

type mockLyricsProvider struct {
	loader *mockPluginLoader
}

func (m *mockLyricsProvider) GetLyrics(ctx context.Context, _ *model.MediaFile) (model.LyricList, error) {
	m.loader.calls++
	_, m.loader.hadDeadline = ctx.Deadline()
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return m.loader.lyrics, m.loader.err
}
