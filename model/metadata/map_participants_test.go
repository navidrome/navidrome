package metadata_test

import (
	"os"

	"github.com/google/uuid"
	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/metadata"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
	"github.com/onsi/gomega/types"
)

var _ = Describe("Participants", func() {
	var (
		props               metadata.Info
		md                  metadata.Metadata
		mf                  model.MediaFile
		mbid1, mbid2, mbid3 string
	)

	BeforeEach(func() {
		_, filePath, _ := tests.TempFile(GinkgoT(), "test", ".mp3")
		fileInfo, _ := os.Stat(filePath)
		mbid1 = uuid.NewString()
		mbid2 = uuid.NewString()
		mbid3 = uuid.NewString()
		props = metadata.Info{
			FileInfo: testFileInfo{fileInfo},
		}
	})

	var toMediaFile = func(tags model.RawTags) model.MediaFile {
		props.Tags = tags
		md = metadata.New("filepath", props)
		return md.ToMediaFile(1, "folderID")
	}

	Describe("ARTIST(S) tags", func() {
		Context("No ARTIST/ARTISTS tags", func() {
			BeforeEach(func() {
				mf = toMediaFile(model.RawTags{})
			})

			It("should set artist to Unknown Artist", func() {
				Expect(mf.Artist).To(Equal("[Unknown Artist]"))
			})

			It("should add an Unknown Artist to participants", func() {
				participants := mf.Participants
				Expect(participants).To(HaveLen(2)) // ARTIST and ALBUMARTIST

				artist := participants[model.RoleArtist][0]
				Expect(artist.ID).ToNot(BeEmpty())
				Expect(artist.Name).To(Equal("[Unknown Artist]"))
				Expect(artist.OrderArtistName).To(Equal("[unknown artist]"))
				Expect(artist.SortArtistName).To(BeEmpty())
				Expect(artist.MbzArtistID).To(BeEmpty())
			})
		})

		Context("Single-valued ARTIST tags, no ARTISTS tags", func() {
			BeforeEach(func() {
				mf = toMediaFile(model.RawTags{
					"ARTIST":               {"Artist Name"},
					"ARTISTSORT":           {"Name, Artist"},
					"MUSICBRAINZ_ARTISTID": {mbid1},
				})
			})

			It("should use the artist tag as display name", func() {
				Expect(mf.Artist).To(Equal("Artist Name"))
			})

			It("should populate the participants", func() {
				participants := mf.Participants
				Expect(participants).To(HaveLen(2)) // ARTIST and ALBUMARTIST
				Expect(participants).To(SatisfyAll(
					HaveKeyWithValue(model.RoleArtist, HaveLen(1)),
				))
				Expect(mf.Artist).To(Equal("Artist Name"))

				artist := participants[model.RoleArtist][0]

				Expect(artist.ID).ToNot(BeEmpty())
				Expect(artist.Name).To(Equal("Artist Name"))
				Expect(artist.OrderArtistName).To(Equal("artist name"))
				Expect(artist.SortArtistName).To(Equal("Name, Artist"))
				Expect(artist.MbzArtistID).To(Equal(mbid1))
			})
		})
		Context("Multiple values in a Single-valued ARTIST tags, no ARTISTS tags", func() {
			BeforeEach(func() {
				mf = toMediaFile(model.RawTags{
					"ARTIST":               {"Artist Name feat. Someone Else"},
					"ARTISTSORT":           {"Name, Artist feat. Else, Someone"},
					"MUSICBRAINZ_ARTISTID": {mbid1},
				})
			})

			It("should split the tag", func() {
				By("keeping the first artist as the display name")
				Expect(mf.Artist).To(Equal("Artist Name feat. Someone Else"))
				Expect(mf.SortArtistName).To(Equal("Name, Artist"))
				Expect(mf.OrderArtistName).To(Equal("artist name"))

				participants := mf.Participants
				Expect(participants).To(SatisfyAll(
					HaveKeyWithValue(model.RoleArtist, HaveLen(2)),
				))

				By("adding the first artist to the participants")
				artist0 := participants[model.RoleArtist][0]
				Expect(artist0.ID).ToNot(BeEmpty())
				Expect(artist0.Name).To(Equal("Artist Name"))
				Expect(artist0.OrderArtistName).To(Equal("artist name"))
				Expect(artist0.SortArtistName).To(Equal("Name, Artist"))

				By("assuming the MBID is for the first artist")
				Expect(artist0.MbzArtistID).To(Equal(mbid1))

				By("adding the second artist to the participants")
				artist1 := participants[model.RoleArtist][1]
				Expect(artist1.ID).ToNot(BeEmpty())
				Expect(artist1.Name).To(Equal("Someone Else"))
				Expect(artist1.OrderArtistName).To(Equal("someone else"))
				Expect(artist1.SortArtistName).To(Equal("Else, Someone"))
				Expect(artist1.MbzArtistID).To(BeEmpty())
			})
			It("should split the tag using case-insensitive separators", func() {
				mf = toMediaFile(model.RawTags{
					"ARTIST": {"A1 FEAT. A2"},
				})
				participants := mf.Participants
				Expect(participants).To(SatisfyAll(
					HaveKeyWithValue(model.RoleArtist, HaveLen(2)),
				))

				artist1 := participants[model.RoleArtist][0]
				Expect(artist1.Name).To(Equal("A1"))
				artist2 := participants[model.RoleArtist][1]
				Expect(artist2.Name).To(Equal("A2"))
			})

			It("should not add an empty artist after split", func() {
				mf = toMediaFile(model.RawTags{
					"ARTIST": {"John Doe /  / Jane Doe"},
				})

				participants := mf.Participants
				Expect(participants).To(HaveKeyWithValue(model.RoleArtist, HaveLen(2)))
				artists := participants[model.RoleArtist]
				Expect(artists[0].Name).To(Equal("John Doe"))
				Expect(artists[1].Name).To(Equal("Jane Doe"))
			})
		})

		Context("Multi-valued ARTIST tags, no ARTISTS tags", func() {
			BeforeEach(func() {
				mf = toMediaFile(model.RawTags{
					"ARTIST":               {"First Artist", "Second Artist"},
					"ARTISTSORT":           {"Name, First Artist", "Name, Second Artist"},
					"MUSICBRAINZ_ARTISTID": {mbid1, mbid2},
				})
			})

			It("should use the first artist name as display name", func() {
				Expect(mf.Artist).To(Equal("First Artist"))
			})

			It("should populate the participants with all artists", func() {
				participants := mf.Participants
				Expect(participants).To(HaveLen(2)) // ARTIST and ALBUMARTIST
				Expect(participants).To(SatisfyAll(
					HaveKeyWithValue(model.RoleArtist, HaveLen(2)),
				))

				artist0 := participants[model.RoleArtist][0]
				Expect(artist0.ID).ToNot(BeEmpty())
				Expect(artist0.Name).To(Equal("First Artist"))
				Expect(artist0.OrderArtistName).To(Equal("first artist"))
				Expect(artist0.SortArtistName).To(Equal("Name, First Artist"))
				Expect(artist0.MbzArtistID).To(Equal(mbid1))

				artist1 := participants[model.RoleArtist][1]
				Expect(artist1.ID).ToNot(BeEmpty())
				Expect(artist1.Name).To(Equal("Second Artist"))
				Expect(artist1.OrderArtistName).To(Equal("second artist"))
				Expect(artist1.SortArtistName).To(Equal("Name, Second Artist"))
				Expect(artist1.MbzArtistID).To(Equal(mbid2))
			})
		})

		Context("Single-valued ARTIST tags, multi-valued ARTISTS tags", func() {
			BeforeEach(func() {
				mf = toMediaFile(model.RawTags{
					"ARTIST":               {"First Artist & Second Artist"},
					"ARTISTSORT":           {"Name, First Artist & Name, Second Artist"},
					"MUSICBRAINZ_ARTISTID": {mbid1, mbid2},
					"ARTISTS":              {"First Artist", "Second Artist"},
					"ARTISTSSORT":          {"Name, First Artist", "Name, Second Artist"},
				})
			})

			It("should use the single-valued tag as display name", func() {
				Expect(mf.Artist).To(Equal("First Artist & Second Artist"))
			})

			It("should prioritize multi-valued tags over single-valued tags", func() {
				participants := mf.Participants
				Expect(participants).To(HaveLen(2)) // ARTIST and ALBUMARTIST
				Expect(participants).To(SatisfyAll(
					HaveKeyWithValue(model.RoleArtist, HaveLen(2)),
				))
				artist0 := participants[model.RoleArtist][0]
				Expect(artist0.ID).ToNot(BeEmpty())
				Expect(artist0.Name).To(Equal("First Artist"))
				Expect(artist0.OrderArtistName).To(Equal("first artist"))
				Expect(artist0.SortArtistName).To(Equal("Name, First Artist"))
				Expect(artist0.MbzArtistID).To(Equal(mbid1))

				artist1 := participants[model.RoleArtist][1]
				Expect(artist1.ID).ToNot(BeEmpty())
				Expect(artist1.Name).To(Equal("Second Artist"))
				Expect(artist1.OrderArtistName).To(Equal("second artist"))
				Expect(artist1.SortArtistName).To(Equal("Name, Second Artist"))
				Expect(artist1.MbzArtistID).To(Equal(mbid2))
			})
		})

		Context("Multi-valued ARTIST tags, multi-valued ARTISTS tags", func() {
			BeforeEach(func() {
				mf = toMediaFile(model.RawTags{
					"ARTIST":               {"First Artist", "Second Artist"},
					"ARTISTSORT":           {"Name, First Artist", "Name, Second Artist"},
					"MUSICBRAINZ_ARTISTID": {mbid1, mbid2},
					"ARTISTS":              {"First Artist 2", "Second Artist 2"},
					"ARTISTSSORT":          {"2, First Artist Name", "2, Second Artist Name"},
				})
			})

			XIt("should use the values concatenated as a display name ", func() {
				Expect(mf.Artist).To(Equal("First Artist + Second Artist"))
			})

			// TODO: remove when the above is implemented
			It("should use the first artist name as display name", func() {
				Expect(mf.Artist).To(Equal("First Artist 2"))
			})

			It("should prioritize ARTISTS tags", func() {
				participants := mf.Participants
				Expect(participants).To(HaveLen(2)) // ARTIST and ALBUMARTIST
				Expect(participants).To(SatisfyAll(
					HaveKeyWithValue(model.RoleArtist, HaveLen(2)),
				))
				artist0 := participants[model.RoleArtist][0]
				Expect(artist0.ID).ToNot(BeEmpty())
				Expect(artist0.Name).To(Equal("First Artist 2"))
				Expect(artist0.OrderArtistName).To(Equal("first artist 2"))
				Expect(artist0.SortArtistName).To(Equal("2, First Artist Name"))
				Expect(artist0.MbzArtistID).To(Equal(mbid1))

				artist1 := participants[model.RoleArtist][1]
				Expect(artist1.ID).ToNot(BeEmpty())
				Expect(artist1.Name).To(Equal("Second Artist 2"))
				Expect(artist1.OrderArtistName).To(Equal("second artist 2"))
				Expect(artist1.SortArtistName).To(Equal("2, Second Artist Name"))
				Expect(artist1.MbzArtistID).To(Equal(mbid2))
			})
		})
	})

	Describe("ALBUMARTIST(S) tags", func() {
		Context("No ALBUMARTIST/ALBUMARTISTS tags", func() {
			When("the COMPILATION tag is not set", func() {
				BeforeEach(func() {
					mf = toMediaFile(model.RawTags{
						"ARTIST":               {"Artist Name"},
						"ARTISTSORT":           {"Name, Artist"},
						"MUSICBRAINZ_ARTISTID": {mbid1},
					})
				})

				It("should use the ARTIST as ALBUMARTIST", func() {
					Expect(mf.AlbumArtist).To(Equal("Artist Name"))
				})

				It("should add the ARTIST to participants as ALBUMARTIST", func() {
					participants := mf.Participants
					Expect(participants).To(HaveLen(2))
					Expect(participants).To(SatisfyAll(
						HaveKeyWithValue(model.RoleAlbumArtist, HaveLen(1)),
					))

					albumArtist := participants[model.RoleAlbumArtist][0]
					Expect(albumArtist.ID).ToNot(BeEmpty())
					Expect(albumArtist.Name).To(Equal("Artist Name"))
					Expect(albumArtist.OrderArtistName).To(Equal("artist name"))
					Expect(albumArtist.SortArtistName).To(Equal("Name, Artist"))
					Expect(albumArtist.MbzArtistID).To(Equal(mbid1))
				})
			})

			When("the COMPILATION tag is true", func() {
				BeforeEach(func() {
					mf = toMediaFile(model.RawTags{
						"COMPILATION": {"1"},
					})
				})

				It("should use the Various Artists as display name", func() {
					Expect(mf.AlbumArtist).To(Equal("Various Artists"))
				})

				It("should add the Various Artists to participants as ALBUMARTIST", func() {
					participants := mf.Participants
					Expect(participants).To(HaveLen(2))
					Expect(participants).To(SatisfyAll(
						HaveKeyWithValue(model.RoleAlbumArtist, HaveLen(1)),
					))

					albumArtist := participants[model.RoleAlbumArtist][0]
					Expect(albumArtist.ID).ToNot(BeEmpty())
					Expect(albumArtist.Name).To(Equal("Various Artists"))
					Expect(albumArtist.OrderArtistName).To(Equal("various artists"))
					Expect(albumArtist.SortArtistName).To(BeEmpty())
					Expect(albumArtist.MbzArtistID).To(Equal(consts.VariousArtistsMbzId))
				})
			})
		})

		Context("ALBUMARTIST tag is set", func() {
			BeforeEach(func() {
				mf = toMediaFile(model.RawTags{
					"ARTIST":                    {"Track Artist Name"},
					"ARTISTSORT":                {"Name, Track Artist"},
					"MUSICBRAINZ_ARTISTID":      {mbid1},
					"ALBUMARTIST":               {"Album Artist Name"},
					"ALBUMARTISTSORT":           {"Album Artist Sort Name"},
					"MUSICBRAINZ_ALBUMARTISTID": {mbid2},
				})
			})

			It("should use the ALBUMARTIST as display name", func() {
				Expect(mf.AlbumArtist).To(Equal("Album Artist Name"))
			})

			It("should populate the participants with the ALBUMARTIST", func() {
				participants := mf.Participants
				Expect(participants).To(HaveLen(2))
				Expect(participants).To(SatisfyAll(
					HaveKeyWithValue(model.RoleAlbumArtist, HaveLen(1)),
				))

				albumArtist := participants[model.RoleAlbumArtist][0]
				Expect(albumArtist.ID).ToNot(BeEmpty())
				Expect(albumArtist.Name).To(Equal("Album Artist Name"))
				Expect(albumArtist.OrderArtistName).To(Equal("album artist name"))
				Expect(albumArtist.SortArtistName).To(Equal("Album Artist Sort Name"))
				Expect(albumArtist.MbzArtistID).To(Equal(mbid2))
			})
		})
	})

	Describe("COMPOSER and LYRICIST tags (with sort names)", func() {
		DescribeTable("should return the correct participation",
			func(role model.Role, nameTag, sortTag string) {
				mf = toMediaFile(model.RawTags{
					nameTag: {"First Name", "Second Name"},
					sortTag: {"Name, First", "Name, Second"},
				})

				participants := mf.Participants
				Expect(participants).To(HaveKeyWithValue(role, HaveLen(2)))

				p := participants[role]
				Expect(p[0].ID).ToNot(BeEmpty())
				Expect(p[0].Name).To(Equal("First Name"))
				Expect(p[0].SortArtistName).To(Equal("Name, First"))
				Expect(p[0].OrderArtistName).To(Equal("first name"))
				Expect(p[1].ID).ToNot(BeEmpty())
				Expect(p[1].Name).To(Equal("Second Name"))
				Expect(p[1].SortArtistName).To(Equal("Name, Second"))
				Expect(p[1].OrderArtistName).To(Equal("second name"))
			},
			Entry("COMPOSER", model.RoleComposer, "COMPOSER", "COMPOSERSORT"),
			Entry("LYRICIST", model.RoleLyricist, "LYRICIST", "LYRICISTSORT"),
		)
	})

	Describe("PERFORMER tags", func() {
		When("PERFORMER tag is set", func() {
			matchPerformer := func(name, orderName, subRole string) types.GomegaMatcher {
				return MatchFields(IgnoreExtras, Fields{
					"Artist": MatchFields(IgnoreExtras, Fields{
						"Name":            Equal(name),
						"OrderArtistName": Equal(orderName),
					}),
					"SubRole": Equal(subRole),
				})
			}

			It("should return the correct participation", func() {
				mf = toMediaFile(model.RawTags{
					"PERFORMER:GUITAR":        {"Eric Clapton", "B.B. King"},
					"PERFORMER:BASS":          {"Nathan East"},
					"PERFORMER:HAMMOND ORGAN": {"Tim Carmon"},
				})

				participants := mf.Participants
				Expect(participants).To(HaveKeyWithValue(model.RolePerformer, HaveLen(4)))

				p := participants[model.RolePerformer]
				Expect(p).To(ContainElements(
					matchPerformer("Eric Clapton", "eric clapton", "Guitar"),
					matchPerformer("B.B. King", "b.b. king", "Guitar"),
					matchPerformer("Nathan East", "nathan east", "Bass"),
					matchPerformer("Tim Carmon", "tim carmon", "Hammond Organ"),
				))
			})
		})
	})

	Describe("Other tags", func() {
		DescribeTable("should return the correct participation",
			func(role model.Role, tag string) {
				mf = toMediaFile(model.RawTags{
					tag: {"John Doe", "Jane Doe"},
				})

				participants := mf.Participants
				Expect(participants).To(HaveKeyWithValue(role, HaveLen(2)))

				p := participants[role]
				Expect(p[0].ID).ToNot(BeEmpty())
				Expect(p[0].Name).To(Equal("John Doe"))
				Expect(p[0].OrderArtistName).To(Equal("john doe"))
				Expect(p[1].ID).ToNot(BeEmpty())
				Expect(p[1].Name).To(Equal("Jane Doe"))
				Expect(p[1].OrderArtistName).To(Equal("jane doe"))
			},
			Entry("CONDUCTOR", model.RoleConductor, "CONDUCTOR"),
			Entry("ARRANGER", model.RoleArranger, "ARRANGER"),
			Entry("PRODUCER", model.RoleProducer, "PRODUCER"),
			Entry("ENGINEER", model.RoleEngineer, "ENGINEER"),
			Entry("MIXER", model.RoleMixer, "MIXER"),
			Entry("REMIXER", model.RoleRemixer, "REMIXER"),
			Entry("DJMIXER", model.RoleDJMixer, "DJMIXER"),
			Entry("DIRECTOR", model.RoleDirector, "DIRECTOR"),
			// TODO PERFORMER
		)
	})

	Describe("Role value splitting", func() {
		When("the tag is single valued", func() {
			It("should split the values by the configured separator", func() {
				mf = toMediaFile(model.RawTags{
					"COMPOSER": {"John Doe/Someone Else/The Album Artist"},
				})

				participants := mf.Participants
				Expect(participants).To(HaveKeyWithValue(model.RoleComposer, HaveLen(3)))
				composers := participants[model.RoleComposer]
				Expect(composers[0].Name).To(Equal("John Doe"))
				Expect(composers[1].Name).To(Equal("Someone Else"))
				Expect(composers[2].Name).To(Equal("The Album Artist"))
			})
			It("should not add an empty participant after split", func() {
				mf = toMediaFile(model.RawTags{
					"COMPOSER": {"John Doe/"},
				})

				participants := mf.Participants
				Expect(participants).To(HaveKeyWithValue(model.RoleComposer, HaveLen(1)))
				composers := participants[model.RoleComposer]
				Expect(composers[0].Name).To(Equal("John Doe"))
			})
			It("should trim the values", func() {
				mf = toMediaFile(model.RawTags{
					"COMPOSER": {"John Doe / Someone Else / The Album Artist"},
				})

				participants := mf.Participants
				Expect(participants).To(HaveKeyWithValue(model.RoleComposer, HaveLen(3)))
				composers := participants[model.RoleComposer]
				Expect(composers[0].Name).To(Equal("John Doe"))
				Expect(composers[1].Name).To(Equal("Someone Else"))
				Expect(composers[2].Name).To(Equal("The Album Artist"))
			})
		})
	})

	Describe("MBID tags", func() {
		It("should set the MBID for the artist based on the track/album artist", func() {
			mf = toMediaFile(model.RawTags{
				"ARTIST":                    {"John Doe", "Jane Doe"},
				"MUSICBRAINZ_ARTISTID":      {mbid1, mbid2},
				"ALBUMARTIST":               {"The Album Artist"},
				"MUSICBRAINZ_ALBUMARTISTID": {mbid3},
				"COMPOSER":                  {"John Doe", "Someone Else", "The Album Artist"},
				"PRODUCER":                  {"Jane Doe", "John Doe"},
			})

			participants := mf.Participants
			Expect(participants).To(HaveKeyWithValue(model.RoleComposer, HaveLen(3)))
			composers := participants[model.RoleComposer]
			Expect(composers[0].MbzArtistID).To(Equal(mbid1))
			Expect(composers[1].MbzArtistID).To(BeEmpty())
			Expect(composers[2].MbzArtistID).To(Equal(mbid3))

			Expect(participants).To(HaveKeyWithValue(model.RoleProducer, HaveLen(2)))
			producers := participants[model.RoleProducer]
			Expect(producers[0].MbzArtistID).To(Equal(mbid2))
			Expect(producers[1].MbzArtistID).To(Equal(mbid1))
		})
	})

	Describe("Non-standard MBID tags", func() {
		var allMappings = map[model.Role]model.TagName{
			model.RoleComposer:  model.TagMusicBrainzComposerID,
			model.RoleLyricist:  model.TagMusicBrainzLyricistID,
			model.RoleConductor: model.TagMusicBrainzConductorID,
			model.RoleArranger:  model.TagMusicBrainzArrangerID,
			model.RoleDirector:  model.TagMusicBrainzDirectorID,
			model.RoleProducer:  model.TagMusicBrainzProducerID,
			model.RoleEngineer:  model.TagMusicBrainzEngineerID,
			model.RoleMixer:     model.TagMusicBrainzMixerID,
			model.RoleRemixer:   model.TagMusicBrainzRemixerID,
			model.RoleDJMixer:   model.TagMusicBrainzDJMixerID,
		}

		It("should handle more artists than mbids", func() {
			for key := range allMappings {
				mf = toMediaFile(map[string][]string{
					key.String():              {"a", "b", "c"},
					allMappings[key].String(): {"f634bf6d-d66a-425d-888a-28ad39392759", "3dfa3c70-d7d3-4b97-b953-c298dd305e12"},
				})

				participants := mf.Participants
				Expect(participants).To(HaveKeyWithValue(key, HaveLen(3)))
				roles := participants[key]

				Expect(roles[0].Name).To(Equal("a"))
				Expect(roles[1].Name).To(Equal("b"))
				Expect(roles[2].Name).To(Equal("c"))

				Expect(roles[0].MbzArtistID).To(Equal("f634bf6d-d66a-425d-888a-28ad39392759"))
				Expect(roles[1].MbzArtistID).To(Equal("3dfa3c70-d7d3-4b97-b953-c298dd305e12"))
				Expect(roles[2].MbzArtistID).To(Equal(""))
			}
		})

		It("should handle more mbids than artists", func() {
			for key := range allMappings {
				mf = toMediaFile(map[string][]string{
					key.String():              {"a", "b"},
					allMappings[key].String(): {"f634bf6d-d66a-425d-888a-28ad39392759", "3dfa3c70-d7d3-4b97-b953-c298dd305e12"},
				})

				participants := mf.Participants
				Expect(participants).To(HaveKeyWithValue(key, HaveLen(2)))
				roles := participants[key]

				Expect(roles[0].Name).To(Equal("a"))
				Expect(roles[1].Name).To(Equal("b"))

				Expect(roles[0].MbzArtistID).To(Equal("f634bf6d-d66a-425d-888a-28ad39392759"))
				Expect(roles[1].MbzArtistID).To(Equal("3dfa3c70-d7d3-4b97-b953-c298dd305e12"))
			}
		})

		It("should refuse duplicate names if no mbid specified", func() {
			for key := range allMappings {
				mf = toMediaFile(map[string][]string{
					key.String(): {"a", "b", "a", "a"},
				})

				participants := mf.Participants
				Expect(participants).To(HaveKeyWithValue(key, HaveLen(2)))
				roles := participants[key]

				Expect(roles[0].Name).To(Equal("a"))
				Expect(roles[0].MbzArtistID).To(Equal(""))
				Expect(roles[1].Name).To(Equal("b"))
				Expect(roles[1].MbzArtistID).To(Equal(""))
			}
		})
	})
})
