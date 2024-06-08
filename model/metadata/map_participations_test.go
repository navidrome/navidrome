package metadata_test

import (
	"os"

	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/metadata"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Participations", func() {
	var (
		props metadata.Info
		md    metadata.Metadata
		mf    model.MediaFile
	)

	BeforeEach(func() {
		_, filePath, _ := tests.TempFile(GinkgoT(), "test", ".mp3")
		fileInfo, _ := os.Stat(filePath)
		props = metadata.Info{
			FileInfo: testFileInfo{fileInfo},
		}
	})

	var toMediaFile = func(tags map[string][]string) model.MediaFile {
		props.Tags = tags
		md = metadata.New("filepath", props)
		return md.ToMediaFile()
	}

	Describe("ARTIST(S) tags", func() {
		Context("No ARTIST/ARTISTS tags", func() {
			BeforeEach(func() {
				mf = toMediaFile(map[string][]string{})
			})

			It("should set artist to Unknown Artist", func() {
				Expect(mf.Artist).To(Equal("[Unknown Artist]"))
			})

			It("should add an Unknown Artist to participations", func() {
				participations := mf.Participations
				Expect(participations).To(HaveLen(1))

				artist := participations[model.RoleArtist][0]
				Expect(artist.ID).ToNot(BeEmpty())
				Expect(artist.Name).To(Equal("[Unknown Artist]"))
				Expect(artist.OrderArtistName).To(Equal("[unknown artist]"))
				Expect(artist.SortArtistName).To(BeEmpty())
				Expect(artist.MbzArtistID).To(BeEmpty())
			})
		})

		Context("Single-valued ARTIST tags, no ARTISTS tags", func() {
			BeforeEach(func() {
				mf = toMediaFile(map[string][]string{
					"ARTIST":               {"Artist Name"},
					"ARTISTSORT":           {"Name, Artist"},
					"MUSICBRAINZ_ARTISTID": {"1234"},
				})
			})

			It("should use the artist tag as display name", func() {
				Expect(mf.Artist).To(Equal("Artist Name"))
			})

			It("should populate the participations", func() {
				participations := mf.Participations
				Expect(participations).To(HaveLen(1))
				Expect(participations).To(SatisfyAll(
					HaveKeyWithValue(model.RoleArtist, HaveLen(1)),
				))
				Expect(mf.Artist).To(Equal("Artist Name"))

				artist := participations[model.RoleArtist][0]

				Expect(artist.ID).ToNot(BeEmpty())
				Expect(artist.Name).To(Equal("Artist Name"))
				Expect(artist.OrderArtistName).To(Equal("artist name"))
				Expect(artist.SortArtistName).To(Equal("Name, Artist"))
				Expect(artist.MbzArtistID).To(Equal("1234"))
			})
		})

		Context("Multi-valued ARTIST tags, no ARTISTS tags", func() {
			BeforeEach(func() {
				mf = toMediaFile(map[string][]string{
					"ARTIST":               {"First Artist", "Second Artist"},
					"ARTISTSORT":           {"Name, First Artist", "Name, Second Artist"},
					"MUSICBRAINZ_ARTISTID": {"1234", "5678"},
				})
			})

			It("should use the first artist name as display name", func() {
				Expect(mf.Artist).To(Equal("First Artist"))
			})

			It("should populate the participations with all artists", func() {
				participations := mf.Participations
				Expect(participations).To(HaveLen(1))
				Expect(participations).To(SatisfyAll(
					HaveKeyWithValue(model.RoleArtist, HaveLen(2)),
				))

				artist0 := participations[model.RoleArtist][0]
				Expect(artist0.ID).ToNot(BeEmpty())
				Expect(artist0.Name).To(Equal("First Artist"))
				Expect(artist0.OrderArtistName).To(Equal("first artist"))
				Expect(artist0.SortArtistName).To(Equal("Name, First Artist"))
				Expect(artist0.MbzArtistID).To(Equal("1234"))

				artist1 := participations[model.RoleArtist][1]
				Expect(artist1.ID).ToNot(BeEmpty())
				Expect(artist1.Name).To(Equal("Second Artist"))
				Expect(artist1.OrderArtistName).To(Equal("second artist"))
				Expect(artist1.SortArtistName).To(Equal("Name, Second Artist"))
				Expect(artist1.MbzArtistID).To(Equal("5678"))
			})
		})

		Context("Single-valued ARTIST tags, multi-valued ARTISTS tags", func() {
			BeforeEach(func() {
				mf = toMediaFile(map[string][]string{
					"ARTIST":               {"First Artist & Second Artist"},
					"ARTISTSORT":           {"Name, First Artist & Name, Second Artist"},
					"MUSICBRAINZ_ARTISTID": {"1234", "5678"},
					"ARTISTS":              {"First Artist", "Second Artist"},
					"ARTISTSSORT":          {"Name, First Artist", "Name, Second Artist"},
				})
			})

			It("should use the single-valued tag as display name", func() {
				Expect(mf.Artist).To(Equal("First Artist & Second Artist"))
			})

			It("should prioritize multi-valued tags over single-valued tags", func() {
				participations := mf.Participations
				Expect(participations).To(HaveLen(1))
				Expect(participations).To(SatisfyAll(
					HaveKeyWithValue(model.RoleArtist, HaveLen(2)),
				))
				artist0 := participations[model.RoleArtist][0]
				Expect(artist0.ID).ToNot(BeEmpty())
				Expect(artist0.Name).To(Equal("First Artist"))
				Expect(artist0.OrderArtistName).To(Equal("first artist"))
				Expect(artist0.SortArtistName).To(Equal("Name, First Artist"))
				Expect(artist0.MbzArtistID).To(Equal("1234"))

				artist1 := participations[model.RoleArtist][1]
				Expect(artist1.ID).ToNot(BeEmpty())
				Expect(artist1.Name).To(Equal("Second Artist"))
				Expect(artist1.OrderArtistName).To(Equal("second artist"))
				Expect(artist1.SortArtistName).To(Equal("Name, Second Artist"))
				Expect(artist1.MbzArtistID).To(Equal("5678"))
			})
		})

		Context("Multi-valued ARTIST tags, multi-valued ARTISTS tags", func() {
			BeforeEach(func() {
				mf = toMediaFile(map[string][]string{
					"ARTIST":               {"First Artist", "Second Artist"},
					"ARTISTSORT":           {"Name, First Artist", "Name, Second Artist"},
					"MUSICBRAINZ_ARTISTID": {"1234", "5678"},
					"ARTISTS":              {"First Artist 2", "Second Artist 2"},
					"ARTISTSSORT":          {"2, First Artist Name", "2, Second Artist Name"},
				})
			})

			It("should use the first artist name as display name", func() {
				Expect(mf.Artist).To(Equal("First Artist 2"))
			})

			It("should prioritize ARTISTS tags", func() {
				participations := mf.Participations
				Expect(participations).To(HaveLen(1))
				Expect(participations).To(SatisfyAll(
					HaveKeyWithValue(model.RoleArtist, HaveLen(2)),
				))
				artist0 := participations[model.RoleArtist][0]
				Expect(artist0.ID).ToNot(BeEmpty())
				Expect(artist0.Name).To(Equal("First Artist 2"))
				Expect(artist0.OrderArtistName).To(Equal("first artist 2"))
				Expect(artist0.SortArtistName).To(Equal("2, First Artist Name"))
				Expect(artist0.MbzArtistID).To(Equal("1234"))

				artist1 := participations[model.RoleArtist][1]
				Expect(artist1.ID).ToNot(BeEmpty())
				Expect(artist1.Name).To(Equal("Second Artist 2"))
				Expect(artist1.OrderArtistName).To(Equal("second artist 2"))
				Expect(artist1.SortArtistName).To(Equal("2, Second Artist Name"))
				Expect(artist1.MbzArtistID).To(Equal("5678"))
			})
		})
	})

	Describe("ALBUMARTIST(S) tags", func() {
		When("there is an ARTIST tag", func() {
			Context("No ALBUMARTIST/ALBUMARTISTS tags", func() {
				BeforeEach(func() {
					mf = toMediaFile(map[string][]string{
						"ARTIST":               {"Artist Name"},
						"ARTISTSORT":           {"Name, Artist"},
						"MUSICBRAINZ_ARTISTID": {"1234"},
					})
				})

				It("should use the ARTIST as ALBUMARTIST", func() {
					Expect(mf.AlbumArtist).To(Equal("Artist Name"))
				})

				It("should add the ARTIST to participations as ALBUMARTIST", func() {
					participations := mf.Participations
					Expect(participations).To(HaveLen(2))
					Expect(participations).To(SatisfyAll(
						HaveKeyWithValue(model.RoleAlbumArtist, HaveLen(1)),
					))

					albumArtist := participations[model.RoleAlbumArtist][0]
					Expect(albumArtist.ID).ToNot(BeEmpty())
					Expect(albumArtist.Name).To(Equal("Artist Name"))
					Expect(albumArtist.OrderArtistName).To(Equal("artist name"))
					Expect(albumArtist.SortArtistName).To(Equal("Name, Artist"))
					Expect(albumArtist.MbzArtistID).To(Equal("1234"))
				})
			})
		})
	})

	XIt("should return the correct participations", func() {
		mf = toMediaFile(map[string][]string{
			"ARTIST":                    {"Artist Name", "Second Artist", "Third Artist"},
			"MUSICBRAINZ_ARTISTID":      {"1234", "5678", "91011"},
			"ALBUMARTIST":               {"Album Artist Name", "Second Album Artist"},
			"ALBUMARTISTSORT":           {"Album Artist Sort Name"},
			"MUSICBRAINZ_ALBUMARTISTID": {"9876"},
			"COMPOSER":                  {"Composer Name", "Second Composer"},
			"CONDUCTOR":                 {"Conductor Name"},
			"ARRANGER":                  {"Arranger Name"},
			"LYRICIST":                  {"Lyricist Name"},
			"PRODUCER":                  {"Producer Name"},
			"ENGINEER":                  {"Engineer Name"},
			"MIXER":                     {"Mixer Name"},
			"REMIXER":                   {"Remixer Name"},
			"DJMIXER":                   {"DJ Mixer Name"},
			"DIRECTOR":                  {"Director Name"},
			//"PERFORMER": {"Performer Name", "Second Performer"},
		})

		participations := mf.Participations
		Expect(participations).To(HaveLen(12))
		Expect(participations).To(SatisfyAll(
			HaveKeyWithValue(model.RoleArtist, HaveLen(3)),
			HaveKeyWithValue(model.RoleAlbumArtist, HaveLen(2)),
			HaveKeyWithValue(model.RoleComposer, HaveLen(2)),
			HaveKeyWithValue(model.RoleConductor, HaveLen(1)),
			HaveKeyWithValue(model.RoleArranger, HaveLen(1)),
			HaveKeyWithValue(model.RoleLyricist, HaveLen(1)),
			HaveKeyWithValue(model.RoleProducer, HaveLen(1)),
			HaveKeyWithValue(model.RoleEngineer, HaveLen(1)),
			HaveKeyWithValue(model.RoleMixer, HaveLen(1)),
			HaveKeyWithValue(model.RoleRemixer, HaveLen(1)),
			HaveKeyWithValue(model.RoleDJMixer, HaveLen(1)),
			HaveKeyWithValue(model.RoleDirector, HaveLen(1)),
		))
		albumArtist0 := participations[model.RoleAlbumArtist][0]
		Expect(albumArtist0.ID).ToNot(BeEmpty())
		Expect(albumArtist0.Name).To(Equal("Album Artist Name"))
		Expect(albumArtist0.OrderArtistName).To(Equal("album artist name"))
		Expect(albumArtist0.SortArtistName).To(Equal("Album Artist Sort Name"))
		Expect(albumArtist0.MbzArtistID).To(Equal("9876"))

		albumArtist1 := participations[model.RoleAlbumArtist][1]
		Expect(albumArtist1.ID).ToNot(BeEmpty())
		Expect(albumArtist1.Name).To(Equal("Second Album Artist"))
		Expect(albumArtist1.OrderArtistName).To(Equal("second album artist"))
		Expect(albumArtist1.SortArtistName).To(BeEmpty())
		Expect(albumArtist1.MbzArtistID).To(BeEmpty())
	})
})
