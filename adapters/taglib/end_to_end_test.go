package taglib

import (
	"io/fs"
	"os"
	"time"

	"github.com/djherbis/times"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/metadata"
	"github.com/navidrome/navidrome/utils/gg"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

type testFileInfo struct {
	fs.FileInfo
}

func (t testFileInfo) BirthTime() time.Time {
	if ts := times.Get(t.FileInfo); ts.HasBirthTime() {
		return ts.BirthTime()
	}
	return t.FileInfo.ModTime()
}

var _ = Describe("Extractor", func() {
	toP := func(name, sortName, mbid string) model.Participant {
		return model.Participant{
			Artist: model.Artist{Name: name, SortArtistName: sortName, MbzArtistID: mbid},
		}
	}

	roles := []struct {
		model.Role
		model.ParticipantList
	}{
		{model.RoleComposer, model.ParticipantList{
			toP("coma a", "a, coma", "bf13b584-f27c-43db-8f42-32898d33d4e2"),
			toP("comb", "comb", "924039a2-09c6-4d29-9b4f-50cc54447d36"),
		}},
		{model.RoleLyricist, model.ParticipantList{
			toP("la a", "a, la", "c84f648f-68a6-40a2-a0cb-d135b25da3c2"),
			toP("lb", "lb", "0a7c582d-143a-4540-b4e9-77200835af65"),
		}},
		{model.RoleArranger, model.ParticipantList{
			toP("aa", "", "4605a1d4-8d15-42a3-bd00-9c20e42f71e6"),
			toP("ab", "", "002f0ff8-77bf-42cc-8216-61a9c43dc145"),
		}},
		{model.RoleConductor, model.ParticipantList{
			toP("cona", "", "af86879b-2141-42af-bad2-389a4dc91489"),
			toP("conb", "", "3dfa3c70-d7d3-4b97-b953-c298dd305e12"),
		}},
		{model.RoleDirector, model.ParticipantList{
			toP("dia", "", "f943187f-73de-4794-be47-88c66f0fd0f4"),
			toP("dib", "", "bceb75da-1853-4b3d-b399-b27f0cafc389"),
		}},
		{model.RoleEngineer, model.ParticipantList{
			toP("ea", "", "f634bf6d-d66a-425d-888a-28ad39392759"),
			toP("eb", "", "243d64ae-d514-44e1-901a-b918d692baee"),
		}},
		{model.RoleProducer, model.ParticipantList{
			toP("pra", "", "d971c8d7-999c-4a5f-ac31-719721ab35d6"),
			toP("prb", "", "f0a09070-9324-434f-a599-6d25ded87b69"),
		}},
		{model.RoleRemixer, model.ParticipantList{
			toP("ra", "", "c7dc6095-9534-4c72-87cc-aea0103462cf"),
			toP("rb", "", "8ebeef51-c08c-4736-992f-c37870becedd"),
		}},
		{model.RoleDJMixer, model.ParticipantList{
			toP("dja", "", "d063f13b-7589-4efc-ab7f-c60e6db17247"),
			toP("djb", "", "3636670c-385f-4212-89c8-0ff51d6bc456"),
		}},
		{model.RoleMixer, model.ParticipantList{
			toP("ma", "", "53fb5a2d-7016-427e-a563-d91819a5f35a"),
			toP("mb", "", "64c13e65-f0da-4ab9-a300-71ee53b0376a"),
		}},
	}

	var e *extractor

	parseTestFile := func(path string) *model.MediaFile {
		mds, err := e.Parse(path)
		Expect(err).ToNot(HaveOccurred())

		info, ok := mds[path]
		Expect(ok).To(BeTrue())

		fileInfo, err := os.Stat(path)
		Expect(err).ToNot(HaveOccurred())
		info.FileInfo = testFileInfo{FileInfo: fileInfo}

		metadata := metadata.New(path, info)
		mf := metadata.ToMediaFile(1, "folderID")
		return &mf
	}

	BeforeEach(func() {
		e = &extractor{}
	})

	Describe("ReplayGain", func() {
		DescribeTable("test replaygain end-to-end", func(file string, trackGain, trackPeak, albumGain, albumPeak *float64) {
			mf := parseTestFile("tests/fixtures/" + file)

			Expect(mf.RGTrackGain).To(Equal(trackGain))
			Expect(mf.RGTrackPeak).To(Equal(trackPeak))
			Expect(mf.RGAlbumGain).To(Equal(albumGain))
			Expect(mf.RGAlbumPeak).To(Equal(albumPeak))
		},
			Entry("mp3 with no replaygain", "no_replaygain.mp3", nil, nil, nil, nil),
			Entry("mp3 with no zero replaygain", "zero_replaygain.mp3", gg.P(0.0), gg.P(1.0), gg.P(0.0), gg.P(1.0)),
		)
	})

	Describe("lyrics", func() {
		makeLyrics := func(code, secondLine string) model.Lyrics {
			return model.Lyrics{
				DisplayArtist: "",
				DisplayTitle:  "",
				Lang:          code,
				Line: []model.Line{
					{Start: gg.P(int64(0)), Value: "This is"},
					{Start: gg.P(int64(2500)), Value: secondLine},
				},
				Offset: nil,
				Synced: true,
			}
		}

		It("should fetch both synced and unsynced lyrics in mixed flac", func() {
			mf := parseTestFile("tests/fixtures/mixed-lyrics.flac")

			lyrics, err := mf.StructuredLyrics()
			Expect(err).ToNot(HaveOccurred())
			Expect(lyrics).To(HaveLen(2))

			Expect(lyrics[0].Synced).To(BeTrue())
			Expect(lyrics[1].Synced).To(BeFalse())
		})

		It("should handle mp3 with uslt and sylt", func() {
			mf := parseTestFile("tests/fixtures/test.mp3")

			lyrics, err := mf.StructuredLyrics()
			Expect(err).ToNot(HaveOccurred())
			Expect(lyrics).To(HaveLen(4))

			engSylt := makeLyrics("eng", "English SYLT")
			engUslt := makeLyrics("eng", "English")
			unsSylt := makeLyrics("xxx", "unspecified SYLT")
			unsUslt := makeLyrics("xxx", "unspecified")

			// Why is the order inconsistent between runs? Nobody knows
			Expect(lyrics).To(Or(
				Equal(model.LyricList{engSylt, engUslt, unsSylt, unsUslt}),
				Equal(model.LyricList{unsSylt, unsUslt, engSylt, engUslt}),
			))
		})

		DescribeTable("format-specific lyrics", func(file string, isId3 bool) {
			mf := parseTestFile("tests/fixtures/" + file)

			lyrics, err := mf.StructuredLyrics()
			Expect(err).To(Not(HaveOccurred()))
			Expect(lyrics).To(HaveLen(2))

			unspec := makeLyrics("xxx", "unspecified")
			eng := makeLyrics("xxx", "English")

			if isId3 {
				eng.Lang = "eng"
			}

			Expect(lyrics).To(Or(
				Equal(model.LyricList{unspec, eng}),
				Equal(model.LyricList{eng, unspec})))
		},
			Entry("flac", "test.flac", false),
			Entry("m4a", "test.m4a", false),
			Entry("ogg", "test.ogg", false),
			Entry("wma", "test.wma", false),
			Entry("wv", "test.wv", false),
			Entry("wav", "test.wav", true),
			Entry("aiff", "test.aiff", true),
		)
	})

	Describe("Participants", func() {
		DescribeTable("test tags consistent across formats", func(format string) {
			mf := parseTestFile("tests/fixtures/test." + format)

			for _, data := range roles {
				role := data.Role
				artists := data.ParticipantList

				actual := mf.Participants[role]
				Expect(actual).To(HaveLen(len(artists)))

				for i := range artists {
					actualArtist := actual[i]
					expectedArtist := artists[i]

					Expect(actualArtist.Name).To(Equal(expectedArtist.Name))
					Expect(actualArtist.SortArtistName).To(Equal(expectedArtist.SortArtistName))
					Expect(actualArtist.MbzArtistID).To(Equal(expectedArtist.MbzArtistID))
				}
			}

			if format != "m4a" {
				performers := mf.Participants[model.RolePerformer]
				Expect(performers).To(HaveLen(8))

				rules := map[string][]string{
					"pgaa": {"2fd0b311-9fa8-4ff9-be5d-f6f3d16b835e", "Guitar"},
					"pgbb": {"223d030b-bf97-4c2a-ad26-b7f7bbe25c93", "Guitar", ""},
					"pvaa": {"cb195f72-448f-41c8-b962-3f3c13d09d38", "Vocals"},
					"pvbb": {"60a1f832-8ca2-49f6-8660-84d57f07b520", "Vocals", "Flute"},
					"pfaa": {"51fb40c-0305-4bf9-a11b-2ee615277725", "", "Flute"},
				}

				for name, rule := range rules {
					mbid := rule[0]
					for i := 1; i < len(rule); i++ {
						found := false

						for _, mapped := range performers {
							if mapped.Name == name && mapped.MbzArtistID == mbid && mapped.SubRole == rule[i] {
								found = true
								break
							}
						}

						Expect(found).To(BeTrue(), "Could not find matching artist")
					}
				}
			}
		},
			Entry("FLAC format", "flac"),
			Entry("M4a format", "m4a"),
			Entry("OGG format", "ogg"),
			Entry("WV format", "wv"),

			Entry("MP3 format", "mp3"),
			Entry("WAV format", "wav"),
			Entry("AIFF format", "aiff"),
		)

		It("should parse wma", func() {
			mf := parseTestFile("tests/fixtures/test.wma")

			for _, data := range roles {
				role := data.Role
				artists := data.ParticipantList
				actual := mf.Participants[role]

				// WMA has no Arranger role
				if role == model.RoleArranger {
					Expect(actual).To(HaveLen(0))
					continue
				}

				Expect(actual).To(HaveLen(len(artists)), role.String())

				// For some bizarre reason, the order is inverted. We also don't get
				// sort names or MBIDs
				for i := range artists {
					idx := len(artists) - 1 - i

					actualArtist := actual[i]
					expectedArtist := artists[idx]

					Expect(actualArtist.Name).To(Equal(expectedArtist.Name))
				}
			}
		})
	})
})
