package lyrics_test

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/conf/configtest"
	"github.com/navidrome/navidrome/core/lyrics"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/tests"
	"github.com/navidrome/navidrome/utils"
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
					Start: ptr(int64(1000)),
					End:   ptr(int64(3000)),
					Value: "Lead words",
					Cue: []model.Cue{
						{
							Start:     ptr(int64(1000)),
							End:       ptr(int64(1500)),
							Value:     "Lead ",
							ByteStart: 0,
							ByteEnd:   4,
						},
						{
							Start:     ptr(int64(1500)),
							End:       ptr(int64(3000)),
							Value:     "words",
							ByteStart: 5,
							ByteEnd:   9,
						},
					},
				},
				{
					Start: ptr(int64(3000)),
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
					Start: ptr(int64(18800)),
					Value: "We're no strangers to love",
				},
				{
					Start: ptr(int64(22800)),
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
					Start: ptr(int64(18800)),
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
					Start: ptr(int64(18800)),
					End:   ptr(int64(22800)),
					Value: "We're from subtitles",
				},
				{
					Start: ptr(int64(22801)),
					End:   ptr(int64(26000)),
					Value: "Another subtitle line",
				},
			},
			Synced: true,
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
		svc := lyrics.NewLyrics(nil)
		list, err := svc.GetLyrics(ctx, &mf)
		Expect(err).To(BeNil())
		Expect(list).To(Equal(expected))
	},
		Entry("embedded > lrc > txt", "embedded,.lrc,.txt", embeddedLyrics),
		Entry("lrc > embedded > txt", ".lrc,embedded,.txt", syncedLyrics),
		Entry("elrc > lrc > embedded", ".elrc,.lrc,embedded", elrcLyrics),
		Entry("srt > txt > embedded", ".srt,.txt,embedded", srtLyrics),
		Entry("txt > lrc > embedded", ".txt,.lrc,embedded", unsyncedLyrics),
		Entry("ttml > elrc > lrc > srt > embedded", ".ttml,.elrc,.lrc,.srt,embedded", ttmlLyrics))

	It("resolves source priority across duplicate media files", func() {
		conf.Server.LyricsPriority = ".ttml,embedded"
		embeddedJSON, err := json.Marshal(embeddedLyrics)
		Expect(err).To(BeNil())

		svc := lyrics.NewLyrics(nil)
		batchSvc, ok := svc.(lyrics.BatchLyrics)
		Expect(ok).To(BeTrue())

		list, err := batchSvc.GetLyricsForMediaFiles(ctx, []model.MediaFile{
			{
				Lyrics: string(embeddedJSON),
				Path:   "tests/fixtures/01 Invisible (RED) Edit Version.mp3",
			},
			{
				Lyrics: "[]",
				Path:   "tests/fixtures/test.mp3",
			},
		})
		Expect(err).To(BeNil())
		Expect(list).To(Equal(ttmlLyrics))
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

		conf.Server.LyricsPriority = ".LRC"
		Expect(os.WriteFile(filepath.Join(dir, "song.LRC"), []byte("[00:01.00]Upper suffix"), 0600)).To(Succeed())

		svc := lyrics.NewLyrics(nil)
		list, err := svc.GetLyrics(ctx, &model.MediaFile{
			LibraryPath: dir,
			Path:        "song.mp3",
		})

		Expect(err).To(BeNil())
		Expect(list).To(HaveLen(1))
		Expect(list[0].Line).To(Equal([]model.Line{
			{Start: ptr(int64(1000)), Value: "Upper suffix"},
		}))
	})

	It("falls through generic YAML sidecars that are not Lyricsfile documents", func() {
		dir, err := os.MkdirTemp("", "lyrics-yaml-fallback-*")
		Expect(err).ToNot(HaveOccurred())
		DeferCleanup(func() {
			Expect(os.RemoveAll(dir)).To(Succeed())
		})

		Expect(os.WriteFile(filepath.Join(dir, "song.yaml"), []byte("title: not lyricsfile\n"), 0600)).To(Succeed())
		Expect(os.WriteFile(filepath.Join(dir, "song.lrc"), []byte("[00:01.00]Fallback line"), 0600)).To(Succeed())

		conf.Server.LyricsPriority = ".yaml,.lrc"
		svc := lyrics.NewLyrics(nil)
		list, err := svc.GetLyrics(ctx, &model.MediaFile{
			LibraryPath: dir,
			Path:        "song.mp3",
		})

		Expect(err).To(BeNil())
		Expect(list).To(HaveLen(1))
		Expect(list[0].Line).To(Equal([]model.Line{
			{Start: ptr(int64(1000)), Value: "Fallback line"},
		}))
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
				conf.Server.LyricsPriority = ".mp3,embedded"

				svc := lyrics.NewLyrics(nil)
				list, err := svc.GetLyrics(ctx, &mf)
				Expect(err).To(BeNil())
				Expect(list).To(Equal(embeddedLyrics))
			})

			It("should return nothing if error happens when trying to parse file", func() {
				conf.Server.LyricsPriority = ".mp3"

				svc := lyrics.NewLyrics(nil)
				list, err := svc.GetLyrics(ctx, &mf)
				Expect(err).To(BeNil())
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
			conf.Server.LyricsPriority = "test-lyrics-plugin"
			mockLoader.lyrics = unsyncedLyrics
			svc := lyrics.NewLyrics(mockLoader)
			list, err := svc.GetLyrics(ctx, &mf)
			Expect(err).To(BeNil())
			Expect(list).To(Equal(unsyncedLyrics))
		})

		It("should try plugin after embedded returns nothing", func() {
			conf.Server.LyricsPriority = "embedded,test-lyrics-plugin"
			mf.Lyrics = "" // No embedded lyrics
			mockLoader.lyrics = unsyncedLyrics
			svc := lyrics.NewLyrics(mockLoader)
			list, err := svc.GetLyrics(ctx, &mf)
			Expect(err).To(BeNil())
			Expect(list).To(Equal(unsyncedLyrics))
		})

		It("should skip plugin if embedded has lyrics", func() {
			conf.Server.LyricsPriority = "embedded,test-lyrics-plugin"
			mockLoader.lyrics = unsyncedLyrics
			svc := lyrics.NewLyrics(mockLoader)
			list, err := svc.GetLyrics(ctx, &mf)
			Expect(err).To(BeNil())
			Expect(list).To(Equal(embeddedLyrics)) // embedded wins
		})

		It("should skip unknown plugin names gracefully", func() {
			conf.Server.LyricsPriority = "nonexistent-plugin,embedded"
			mockLoader.notFound = true
			svc := lyrics.NewLyrics(mockLoader)
			list, err := svc.GetLyrics(ctx, &mf)
			Expect(err).To(BeNil())
			Expect(list).To(Equal(embeddedLyrics)) // falls through to embedded
		})

		It("should preserve plugin name case from config", func() {
			conf.Server.LyricsPriority = "MyLyricsPlugin"
			mockLoader.pluginName = "MyLyricsPlugin"
			mockLoader.lyrics = unsyncedLyrics
			svc := lyrics.NewLyrics(mockLoader)
			list, err := svc.GetLyrics(ctx, &mf)
			Expect(err).To(BeNil())
			Expect(list).To(Equal(unsyncedLyrics))
		})

		It("should handle plugin error gracefully", func() {
			conf.Server.LyricsPriority = "test-lyrics-plugin,embedded"
			mockLoader.err = fmt.Errorf("plugin error")
			svc := lyrics.NewLyrics(mockLoader)
			list, err := svc.GetLyrics(ctx, &mf)
			Expect(err).To(BeNil())
			Expect(list).To(Equal(embeddedLyrics)) // falls through to embedded
		})
	})
})

type mockPluginLoader struct {
	lyrics     model.LyricList
	err        error
	notFound   bool
	pluginName string // expected plugin name (exact match, like real manager)
}

func (m *mockPluginLoader) PluginNames(_ string) []string {
	if m.notFound {
		return nil
	}
	return []string{"test-lyrics-plugin"}
}

func (m *mockPluginLoader) LoadLyricsProvider(name string) (lyrics.Lyrics, bool) {
	if m.notFound {
		return nil, false
	}
	// If pluginName is set, require exact match (like the real plugin manager)
	if m.pluginName != "" && name != m.pluginName {
		return nil, false
	}
	return &mockLyricsProvider{lyrics: m.lyrics, err: m.err}, true
}

type mockLyricsProvider struct {
	lyrics model.LyricList
	err    error
}

func (m *mockLyricsProvider) GetLyrics(_ context.Context, _ *model.MediaFile) (model.LyricList, error) {
	return m.lyrics, m.err
}
