package subsonic

import (
	"encoding/json"
	"path/filepath"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/conf/configtest"
	"github.com/navidrome/navidrome/core/lyrics"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/server/subsonic/responses"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("GetLyricsBySongId", func() {
	var router *Router
	var ds model.DataStore
	mockRepo := &mockedMediaFile{MockMediaFileRepo: tests.MockMediaFileRepo{}}

	BeforeEach(func() {
		ds = &tests.MockDataStore{
			MockedMediaFile: mockRepo,
		}
		router = New(ds, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, lyrics.NewLyrics(ds, nil), nil, nil)
		DeferCleanup(configtest.SetupConfig())
		conf.Server.LyricsPriority = "embedded,.lrc"
	})

	const syncedLyrics = "[00:18.80]We're no strangers to love\n[00:22.801]You know the rules and so do I"
	const unsyncedLyrics = "We're no strangers to love\nYou know the rules and so do I"
	const metadata = "[ar:Rick Astley]\n[ti:That one song]\n[offset:-100]"
	var times = []int64{18800, 22801}

	compareResponses := func(actual *responses.LyricsList, expected responses.LyricsList) {
		Expect(actual).ToNot(BeNil())
		Expect(actual.StructuredLyrics).To(HaveLen(len(expected.StructuredLyrics)))
		for i, realLyric := range actual.StructuredLyrics {
			expectedLyric := expected.StructuredLyrics[i]

			Expect(realLyric.DisplayArtist).To(Equal(expectedLyric.DisplayArtist))
			Expect(realLyric.DisplayTitle).To(Equal(expectedLyric.DisplayTitle))
			Expect(realLyric.Kind).To(Equal(expectedLyric.Kind))
			Expect(realLyric.Lang).To(Equal(expectedLyric.Lang))
			Expect(realLyric.Synced).To(Equal(expectedLyric.Synced))
			Expect(realLyric.Agents).To(Equal(expectedLyric.Agents))

			if expectedLyric.Offset == nil {
				Expect(realLyric.Offset).To(BeNil())
			} else {
				Expect(*realLyric.Offset).To(Equal(*expectedLyric.Offset))
			}

			Expect(realLyric.Line).To(HaveLen(len(expectedLyric.Line)))
			for j, realLine := range realLyric.Line {
				expectedLine := expectedLyric.Line[j]
				Expect(realLine.Value).To(Equal(expectedLine.Value))

				if expectedLine.Start == nil {
					Expect(realLine.Start).To(BeNil())
				} else {
					Expect(*realLine.Start).To(Equal(*expectedLine.Start))
				}
			}

			Expect(realLyric.CueLine).To(HaveLen(len(expectedLyric.CueLine)))
			for j, realCueLine := range realLyric.CueLine {
				expectedCueLine := expectedLyric.CueLine[j]
				Expect(realCueLine.Index).To(Equal(expectedCueLine.Index))
				Expect(realCueLine.Value).To(Equal(expectedCueLine.Value))
				Expect(realCueLine.AgentID).To(Equal(expectedCueLine.AgentID))
				if expectedCueLine.Start == nil {
					Expect(realCueLine.Start).To(BeNil())
				} else {
					Expect(*realCueLine.Start).To(Equal(*expectedCueLine.Start))
				}
				if expectedCueLine.End == nil {
					Expect(realCueLine.End).To(BeNil())
				} else {
					Expect(*realCueLine.End).To(Equal(*expectedCueLine.End))
				}

				Expect(realCueLine.Cue).To(HaveLen(len(expectedCueLine.Cue)))
				for k, realCue := range realCueLine.Cue {
					expectedCue := expectedCueLine.Cue[k]
					Expect(realCue.Value).To(Equal(expectedCue.Value))
					Expect(realCue.Start).To(Equal(expectedCue.Start))
					Expect(realCue.ByteStart).To(Equal(expectedCue.ByteStart))
					Expect(realCue.ByteEnd).To(Equal(expectedCue.ByteEnd))
					if expectedCue.End == nil {
						Expect(realCue.End).To(BeNil())
					} else {
						Expect(*realCue.End).To(Equal(*expectedCue.End))
					}
				}
			}
		}
	}

	It("should return mixed lyrics", func() {
		r := newGetRequest("id=1")
		syncedList, _ := model.ParseLyrics(GinkgoT().Context(), ".lrc", "eng", []byte(syncedLyrics))
		unsyncedList, _ := model.ParseLyrics(GinkgoT().Context(), ".lrc", "xxx", []byte(unsyncedLyrics))
		synced, _ := syncedList.Main()
		unsynced, _ := unsyncedList.Main()
		lyricsJson, err := json.Marshal(model.LyricList{
			synced, unsynced,
		})
		Expect(err).ToNot(HaveOccurred())

		mockRepo.SetData(model.MediaFiles{
			{
				ID:     "1",
				Artist: "Rick Astley",
				Title:  "Never Gonna Give You Up",
				Lyrics: string(lyricsJson),
			},
		})

		response, err := router.GetLyricsBySongId(r)
		Expect(err).ToNot(HaveOccurred())
		compareResponses(response.LyricsList, responses.LyricsList{
			StructuredLyrics: responses.StructuredLyrics{
				{
					Lang:          "eng",
					DisplayArtist: "Rick Astley",
					DisplayTitle:  "Never Gonna Give You Up",
					Synced:        true,
					Line: []responses.Line{
						{
							Start: &times[0],
							Value: "We're no strangers to love",
						},
						{
							Start: &times[1],
							Value: "You know the rules and so do I",
						},
					},
				},
				{
					Lang:          "xxx",
					DisplayArtist: "Rick Astley",
					DisplayTitle:  "Never Gonna Give You Up",
					Synced:        false,
					Line: []responses.Line{
						{
							Value: "We're no strangers to love",
						},
						{
							Value: "You know the rules and so do I",
						},
					},
				},
			},
		})
	})

	It("should parse lrc metadata", func() {
		r := newGetRequest("id=1")
		syncedList, _ := model.ParseLyrics(GinkgoT().Context(), ".lrc", "eng", []byte(metadata+"\n"+syncedLyrics))
		synced, _ := syncedList.Main()
		lyricsJson, err := json.Marshal(model.LyricList{
			synced,
		})
		Expect(err).ToNot(HaveOccurred())
		mockRepo.SetData(model.MediaFiles{
			{
				ID:     "1",
				Artist: "Rick Astley",
				Title:  "Never Gonna Give You Up",
				Lyrics: string(lyricsJson),
			},
		})

		response, err := router.GetLyricsBySongId(r)
		Expect(err).ToNot(HaveOccurred())

		compareResponses(response.LyricsList, responses.LyricsList{
			StructuredLyrics: responses.StructuredLyrics{
				{
					DisplayArtist: "Rick Astley",
					DisplayTitle:  "That one song",
					Lang:          "eng",
					Synced:        true,
					Line: []responses.Line{
						{
							Start: &times[0],
							Value: "We're no strangers to love",
						},
						{
							Start: &times[1],
							Value: "You know the rules and so do I",
						},
					},
					Offset: new(int64(-100)),
				},
			},
		})
	})

	It("should return multilingual TTML sidecar lyrics", func() {
		conf.Server.LyricsPriority = ".ttml,embedded"
		r := newGetRequest("id=1")

		fixturesDir, err := filepath.Abs("tests/fixtures")
		Expect(err).ToNot(HaveOccurred())
		mockRepo.SetData(model.MediaFiles{
			{
				ID:          "1",
				LibraryPath: fixturesDir,
				Path:        "test.mp3",
				Artist:      "Rick Astley",
				Title:       "Never Gonna Give You Up",
				Lyrics:      "[]",
			},
		})

		response, err := router.GetLyricsBySongId(r)
		Expect(err).ToNot(HaveOccurred())

		porTime := int64(18800)
		ttmlTime := int64(22800)
		compareResponses(response.LyricsList, responses.LyricsList{
			StructuredLyrics: responses.StructuredLyrics{
				{
					DisplayArtist: "Rick Astley",
					DisplayTitle:  "Never Gonna Give You Up",
					Lang:          "eng",
					Synced:        true,
					Line: []responses.Line{
						{
							Start: &times[0],
							Value: "We're no strangers to love",
						},
						{
							Start: &ttmlTime,
							Value: "You know the rules and so do I",
						},
					},
				},
				{
					DisplayArtist: "Rick Astley",
					DisplayTitle:  "Never Gonna Give You Up",
					Lang:          "por",
					Synced:        true,
					Line: []responses.Line{
						{
							Start: &porTime,
							Value: "Nao somos estranhos ao amor",
						},
					},
				},
			},
		})
	})

	It("should return metadata-linked translation and pronunciation tracks from TTML", func() {
		conf.Server.LyricsPriority = ".ttml,embedded"
		r := newGetRequest("id=1&enhanced=true")

		fixturesDir, err := filepath.Abs("tests/fixtures")
		Expect(err).ToNot(HaveOccurred())
		mockRepo.SetData(model.MediaFiles{
			{
				ID:          "1",
				LibraryPath: fixturesDir,
				Path:        "test-metadata.mp3",
				Artist:      "Rick Astley",
				Title:       "Never Gonna Give You Up",
				Lyrics:      "[]",
			},
		})

		response, err := router.GetLyricsBySongId(r)
		Expect(err).ToNot(HaveOccurred())

		mainStartA := int64(1000)
		mainStartB := int64(2000)
		tokenStartA := int64(2000)
		tokenEndA := int64(2300)
		tokenStartB := int64(2300)
		tokenEndB := int64(2600)
		compareResponses(response.LyricsList, responses.LyricsList{
			StructuredLyrics: responses.StructuredLyrics{
				{
					DisplayArtist: "Rick Astley",
					DisplayTitle:  "Never Gonna Give You Up",
					Kind:          "main",
					Lang:          "ja",
					Synced:        true,
					Line: []responses.Line{
						{
							Start: &mainStartA,
							Value: "こんにちは",
						},
						{
							Start: &mainStartB,
							Value: "こんばんは",
						},
					},
				},
				{
					DisplayArtist: "Rick Astley",
					DisplayTitle:  "Never Gonna Give You Up",
					Kind:          "translation",
					Lang:          "es",
					Synced:        true,
					Line: []responses.Line{
						{
							Start: &mainStartA,
							Value: "Hola",
						},
					},
				},
				{
					DisplayArtist: "Rick Astley",
					DisplayTitle:  "Never Gonna Give You Up",
					Kind:          "pronunciation",
					Lang:          "ja-latn",
					Synced:        true,
					Line: []responses.Line{
						{
							Start: &mainStartB,
							Value: "konni",
						},
					},
					CueLine: []responses.CueLine{
						{
							Index: 0,
							Start: &mainStartB,
							End:   &tokenEndB,
							Value: "konni",
							Cue: []responses.LyricCue{
								{
									Start:     tokenStartA,
									End:       &tokenEndA,
									ByteStart: 0,
									ByteEnd:   1,
									Value:     "ko",
								},
								{
									Start:     tokenStartB,
									End:       &tokenEndB,
									ByteStart: 2,
									ByteEnd:   4,
									Value:     "nni",
								},
							},
						},
					},
				},
			},
		})
	})

	It("should return cue lines for songLyrics v2 clients with enhanced=true", func() {
		r := newGetRequest("id=1&enhanced=true")

		lineStart := int64(1000)
		lineEnd := int64(3000)
		tokenStartA := int64(1000)
		tokenEndA := int64(1400)
		tokenStartB := int64(2000)
		tokenEndB := int64(2500)
		lyricsJson, err := json.Marshal(model.LyricList{
			{
				Lang:   "eng",
				Agents: []model.Agent{{ID: "lead", Role: "main"}, {ID: "__nd_bg__|lead", Role: "bg"}},
				Synced: true,
				Line: []model.Line{
					{
						Start: &lineStart,
						End:   &lineEnd,
						Value: "Hello echo",
						Cue: []model.Cue{
							{
								Start:     &tokenStartA,
								End:       &tokenEndA,
								Value:     "Hello",
								ByteStart: 0,
								ByteEnd:   4,
								AgentID:   "lead",
							},
							{
								Start:     &tokenStartB,
								End:       &tokenEndB,
								Value:     "echo",
								ByteStart: 6,
								ByteEnd:   9,
								AgentID:   "__nd_bg__|lead",
							},
						},
					},
				},
			},
		})
		Expect(err).ToNot(HaveOccurred())

		mockRepo.SetData(model.MediaFiles{
			{
				ID:     "1",
				Artist: "Rick Astley",
				Title:  "Never Gonna Give You Up",
				Lyrics: string(lyricsJson),
			},
		})

		response, err := router.GetLyricsBySongId(r)
		Expect(err).ToNot(HaveOccurred())
		compareResponses(response.LyricsList, responses.LyricsList{
			StructuredLyrics: responses.StructuredLyrics{
				{
					DisplayArtist: "Rick Astley",
					DisplayTitle:  "Never Gonna Give You Up",
					Kind:          "main",
					Lang:          "eng",
					Synced:        true,
					Agents: []responses.Agent{
						{ID: "lead", Role: "main"},
						{ID: "__nd_bg__|lead", Role: "bg"},
					},
					Line: []responses.Line{
						{
							Start: &lineStart,
							Value: "Hello echo",
						},
					},
					CueLine: []responses.CueLine{
						{
							Index:   0,
							Start:   &lineStart,
							End:     &lineEnd,
							Value:   "Hello",
							AgentID: "lead",
							Cue: []responses.LyricCue{
								{
									Start:     tokenStartA,
									End:       &tokenEndA,
									ByteStart: 0,
									ByteEnd:   4,
									Value:     "Hello",
								},
							},
						},
						{
							Index:   0,
							Start:   &lineStart,
							End:     &lineEnd,
							Value:   "echo",
							AgentID: "__nd_bg__|lead",
							Cue: []responses.LyricCue{
								{
									Start:     tokenStartB,
									End:       &tokenEndB,
									ByteStart: 0,
									ByteEnd:   3,
									Value:     "echo",
								},
							},
						},
					},
				},
			},
		})
	})

	It("should preserve shared edge text when remapping agent cue lines", func() {
		lineStart := int64(1000)
		lineEnd := int64(2000)
		cueStart := int64(1200)
		cueEnd := int64(1800)

		cueLines := buildCueLines(model.Line{
			Start: &lineStart,
			End:   &lineEnd,
			Value: "(Hello)",
			Cue: []model.Cue{
				{
					Start:     &cueStart,
					End:       &cueEnd,
					Value:     "Hello",
					ByteStart: 1,
					ByteEnd:   5,
					AgentID:   "lead",
				},
				{
					Start:     &cueStart,
					End:       &cueEnd,
					Value:     "Hello",
					ByteStart: 1,
					ByteEnd:   5,
					AgentID:   "__nd_bg__|lead",
				},
			},
		}, 0, newLyricAgents([]model.Agent{
			{ID: "lead", Role: "main"},
			{ID: "__nd_bg__|lead", Role: "bg"},
		}))

		Expect(cueLines).To(Equal([]responses.CueLine{
			{
				Index:   0,
				Start:   &lineStart,
				End:     &lineEnd,
				Value:   "(Hello)",
				AgentID: "lead",
				Cue: []responses.LyricCue{
					{
						Start:     cueStart,
						End:       &cueEnd,
						Value:     "Hello",
						ByteStart: 1,
						ByteEnd:   5,
					},
				},
			},
			{
				Index:   0,
				Start:   &lineStart,
				End:     &lineEnd,
				Value:   "(Hello)",
				AgentID: "__nd_bg__|lead",
				Cue: []responses.LyricCue{
					{
						Start:     cueStart,
						End:       &cueEnd,
						Value:     "Hello",
						ByteStart: 1,
						ByteEnd:   5,
					},
				},
			},
		}))
	})

	It("should remap cue offsets for interleaved agent cue lines", func() {
		r := newGetRequest("id=1&enhanced=true")

		lineStart := int64(82889)
		lineEnd := int64(86859)
		realStart := int64(85593)
		realEnd := int64(85934)
		slowStart := int64(85934)
		slowEnd := int64(86751)
		bgStartA := int64(83881)
		bgEndA := int64(84243)
		bgStartB := int64(86232)
		bgEndB := int64(86859)
		lyricsJSON, err := json.Marshal(model.LyricList{
			{
				Lang:   "eng",
				Agents: []model.Agent{{ID: "v2", Role: "main"}, {ID: "__nd_bg__|v2", Role: "bg"}},
				Synced: true,
				Line: []model.Line{
					{
						Start: &lineStart,
						End:   &lineEnd,
						Value: "real slow (When you slide)",
						Cue: []model.Cue{
							{
								Start:     &realStart,
								End:       &realEnd,
								Value:     "real",
								ByteStart: 0,
								ByteEnd:   3,
								AgentID:   "v2",
							},
							{
								Start:     &slowStart,
								End:       &slowEnd,
								Value:     "slow",
								ByteStart: 5,
								ByteEnd:   8,
								AgentID:   "v2",
							},
							{
								Start:     &bgStartA,
								End:       &bgEndA,
								Value:     "(When you",
								ByteStart: 10,
								ByteEnd:   18,
								AgentID:   "__nd_bg__|v2",
							},
							{
								Start:     &bgStartB,
								End:       &bgEndB,
								Value:     "slide)",
								ByteStart: 20,
								ByteEnd:   25,
								AgentID:   "__nd_bg__|v2",
							},
						},
					},
				},
			},
		})
		Expect(err).ToNot(HaveOccurred())

		mockRepo.SetData(model.MediaFiles{
			{
				ID:     "1",
				Artist: "Rick Astley",
				Title:  "Never Gonna Give You Up",
				Lyrics: string(lyricsJSON),
			},
		})

		response, err := router.GetLyricsBySongId(r)
		Expect(err).ToNot(HaveOccurred())
		compareResponses(response.LyricsList, responses.LyricsList{
			StructuredLyrics: responses.StructuredLyrics{
				{
					DisplayArtist: "Rick Astley",
					DisplayTitle:  "Never Gonna Give You Up",
					Kind:          "main",
					Lang:          "eng",
					Synced:        true,
					Agents: []responses.Agent{
						{ID: "v2", Role: "main"},
						{ID: "__nd_bg__|v2", Role: "bg"},
					},
					Line: []responses.Line{
						{
							Start: &lineStart,
							Value: "real slow (When you slide)",
						},
					},
					CueLine: []responses.CueLine{
						{
							Index:   0,
							Start:   &lineStart,
							End:     &lineEnd,
							Value:   "real slow",
							AgentID: "v2",
							Cue: []responses.LyricCue{
								{
									Start:     realStart,
									End:       &realEnd,
									ByteStart: 0,
									ByteEnd:   3,
									Value:     "real",
								},
								{
									Start:     slowStart,
									End:       &slowEnd,
									ByteStart: 5,
									ByteEnd:   8,
									Value:     "slow",
								},
							},
						},
						{
							Index:   0,
							Start:   &lineStart,
							End:     &lineEnd,
							Value:   "(When you slide)",
							AgentID: "__nd_bg__|v2",
							Cue: []responses.LyricCue{
								{
									Start:     bgStartA,
									End:       &bgEndA,
									ByteStart: 0,
									ByteEnd:   8,
									Value:     "(When you",
								},
								{
									Start:     bgStartB,
									End:       &bgEndB,
									ByteStart: 10,
									ByteEnd:   15,
									Value:     "slide)",
								},
							},
						},
					},
				},
			},
		})
	})

	It("should keep enhanced line-level lyrics when no cue data is available", func() {
		r := newGetRequest("id=1&enhanced=true")

		lineStart := int64(1000)
		lineEnd := int64(3000)
		lyricsJSON, err := json.Marshal(model.LyricList{
			{
				Kind:   "main",
				Lang:   "eng",
				Synced: true,
				Line: []model.Line{
					{
						Start: &lineStart,
						End:   &lineEnd,
						Value: "Line without word timing",
					},
				},
			},
		})
		Expect(err).ToNot(HaveOccurred())

		mockRepo.SetData(model.MediaFiles{
			{
				ID:     "1",
				Artist: "Rick Astley",
				Title:  "Never Gonna Give You Up",
				Lyrics: string(lyricsJSON),
			},
		})

		response, err := router.GetLyricsBySongId(r)
		Expect(err).ToNot(HaveOccurred())
		compareResponses(response.LyricsList, responses.LyricsList{
			StructuredLyrics: responses.StructuredLyrics{
				{
					DisplayArtist: "Rick Astley",
					DisplayTitle:  "Never Gonna Give You Up",
					Kind:          "main",
					Lang:          "eng",
					Synced:        true,
					Line: []responses.Line{
						{
							Start: &lineStart,
							Value: "Line without word timing",
						},
					},
				},
			},
		})
	})

	It("should return required cue byte offsets for ambiguous and multibyte cue lines", func() {
		r := newGetRequest("id=1&enhanced=true")

		asciiLineStart := int64(0)
		asciiLineEnd := int64(2400)
		asciiCueStartA := int64(0)
		asciiCueEndA := int64(300)
		asciiCueStartB := int64(900)
		asciiCueEndB := int64(1300)
		asciiCueStartC := int64(1300)
		asciiCueEndC := int64(1600)
		asciiCueStartD := int64(1600)

		utfLineStart := int64(2747)
		utfLineEnd := int64(6214)
		utfCueStartA := int64(2747)
		utfCueEndA := int64(3018)
		utfCueStartB := int64(3018)
		utfCueEndB := int64(3179)
		utfCueStartC := int64(3582)
		utfCueEndC := int64(4100)
		utfCueStartD := int64(4500)
		utfCueEndD := int64(6214)

		lyricsJSON, err := json.Marshal(model.LyricList{
			{
				Lang:   "eng",
				Synced: true,
				Line: []model.Line{
					{
						Start: &asciiLineStart,
						End:   &asciiLineEnd,
						Value: "Oh love love me tonight",
						Cue: []model.Cue{
							{Start: &asciiCueStartA, End: &asciiCueEndA, Value: "Oh", ByteStart: 0, ByteEnd: 1},
							{Start: &asciiCueStartB, End: &asciiCueEndB, Value: "love", ByteStart: 8, ByteEnd: 11},
							{Start: &asciiCueStartC, End: &asciiCueEndC, Value: "me", ByteStart: 13, ByteEnd: 14},
							{Start: &asciiCueStartD, Value: "tonight", ByteStart: 16, ByteEnd: 22},
						},
					},
					{
						Start: &utfLineStart,
						End:   &utfLineEnd,
						Value: "눈을 뜬 순간",
						Cue: []model.Cue{
							{Start: &utfCueStartA, End: &utfCueEndA, Value: "눈", ByteStart: 0, ByteEnd: 2},
							{Start: &utfCueStartB, End: &utfCueEndB, Value: "을", ByteStart: 3, ByteEnd: 5},
							{Start: &utfCueStartC, End: &utfCueEndC, Value: "뜬", ByteStart: 7, ByteEnd: 9},
							{Start: &utfCueStartD, End: &utfCueEndD, Value: "순간", ByteStart: 11, ByteEnd: 16},
						},
					},
				},
			},
		})
		Expect(err).ToNot(HaveOccurred())

		mockRepo.SetData(model.MediaFiles{
			{
				ID:     "1",
				Artist: "Rick Astley",
				Title:  "Never Gonna Give You Up",
				Lyrics: string(lyricsJSON),
			},
		})

		response, err := router.GetLyricsBySongId(r)
		Expect(err).ToNot(HaveOccurred())
		compareResponses(response.LyricsList, responses.LyricsList{
			StructuredLyrics: responses.StructuredLyrics{
				{
					DisplayArtist: "Rick Astley",
					DisplayTitle:  "Never Gonna Give You Up",
					Kind:          "main",
					Lang:          "eng",
					Synced:        true,
					Line: []responses.Line{
						{Start: &asciiLineStart, Value: "Oh love love me tonight"},
						{Start: &utfLineStart, Value: "눈을 뜬 순간"},
					},
					CueLine: []responses.CueLine{
						{
							Index: 0,
							Start: &asciiLineStart,
							End:   &asciiLineEnd,
							Value: "Oh love love me tonight",
							Cue: []responses.LyricCue{
								{Start: asciiCueStartA, End: &asciiCueEndA, Value: "Oh", ByteStart: 0, ByteEnd: 1},
								{Start: asciiCueStartB, End: &asciiCueEndB, Value: "love", ByteStart: 8, ByteEnd: 11},
								{Start: asciiCueStartC, End: &asciiCueEndC, Value: "me", ByteStart: 13, ByteEnd: 14},
								{Start: asciiCueStartD, End: &asciiLineEnd, Value: "tonight", ByteStart: 16, ByteEnd: 22},
							},
						},
						{
							Index: 1,
							Start: &utfLineStart,
							End:   &utfLineEnd,
							Value: "눈을 뜬 순간",
							Cue: []responses.LyricCue{
								{Start: utfCueStartA, End: &utfCueEndA, Value: "눈", ByteStart: 0, ByteEnd: 2},
								{Start: utfCueStartB, End: &utfCueEndB, Value: "을", ByteStart: 3, ByteEnd: 5},
								{Start: utfCueStartC, End: &utfCueEndC, Value: "뜬", ByteStart: 7, ByteEnd: 9},
								{Start: utfCueStartD, End: &utfCueEndD, Value: "순간", ByteStart: 11, ByteEnd: 16},
							},
						},
					},
				},
			},
		})
	})
})
