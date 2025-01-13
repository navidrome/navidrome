package metadata

import (
	"strings"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/conf/configtest"
	"github.com/navidrome/navidrome/model"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("getPID", func() {
	var (
		md     Metadata
		mf     model.MediaFile
		sum    hashFunc
		getPID func(mf model.MediaFile, md Metadata, spec string) string
	)

	BeforeEach(func() {
		sum = func(s ...string) string { return "(" + strings.Join(s, ",") + ")" }
		getPID = createGetPID(sum)
	})

	Context("attributes are tags", func() {
		spec := "musicbrainz_trackid|album,discnumber,tracknumber"
		When("no attributes were present", func() {
			It("should return empty pid", func() {
				md.tags = map[model.TagName][]string{}
				pid := getPID(mf, md, spec)
				Expect(pid).To(Equal("()"))
			})
		})
		When("all fields are present", func() {
			It("should return the pid", func() {
				md.tags = map[model.TagName][]string{
					"musicbrainz_trackid": {"mbtrackid"},
					"album":               {"album name"},
					"discnumber":          {"1"},
					"tracknumber":         {"1"},
				}
				Expect(getPID(mf, md, spec)).To(Equal("(mbtrackid)"))
			})
		})
		When("only first field is present", func() {
			It("should return the pid", func() {
				md.tags = map[model.TagName][]string{
					"musicbrainz_trackid": {"mbtrackid"},
				}
				Expect(getPID(mf, md, spec)).To(Equal("(mbtrackid)"))
			})
		})
		When("first is empty, but second field is present", func() {
			It("should return the pid", func() {
				md.tags = map[model.TagName][]string{
					"album":      {"album name"},
					"discnumber": {"1"},
				}
				Expect(getPID(mf, md, spec)).To(Equal("(album name\\1\\)"))
			})
		})
	})
	Context("calculated attributes", func() {
		BeforeEach(func() {
			DeferCleanup(configtest.SetupConfig())
			conf.Server.PID.Album = "musicbrainz_albumid|albumartistid,album,version,releasedate"
		})
		When("field is title", func() {
			It("should return the pid", func() {
				spec := "title|folder"
				md.tags = map[model.TagName][]string{"title": {"title"}}
				md.filePath = "/path/to/file.mp3"
				mf.Title = "Title"
				Expect(getPID(mf, md, spec)).To(Equal("(Title)"))
			})
		})
		When("field is folder", func() {
			It("should return the pid", func() {
				spec := "folder|title"
				md.tags = map[model.TagName][]string{"title": {"title"}}
				mf.Path = "/path/to/file.mp3"
				Expect(getPID(mf, md, spec)).To(Equal("(/path/to)"))
			})
		})
		When("field is albumid", func() {
			It("should return the pid", func() {
				spec := "albumid|title"
				md.tags = map[model.TagName][]string{
					"title":       {"title"},
					"album":       {"album name"},
					"version":     {"version"},
					"releasedate": {"2021-01-01"},
				}
				mf.AlbumArtist = "Album Artist"
				Expect(getPID(mf, md, spec)).To(Equal("(((album artist)\\album name\\version\\2021-01-01))"))
			})
		})
		When("field is albumartistid", func() {
			It("should return the pid", func() {
				spec := "musicbrainz_albumartistid|albumartistid"
				md.tags = map[model.TagName][]string{
					"albumartist": {"Album Artist"},
				}
				mf.AlbumArtist = "Album Artist"
				Expect(getPID(mf, md, spec)).To(Equal("((album artist))"))
			})
		})
		When("field is album", func() {
			It("should return the pid", func() {
				spec := "album|title"
				md.tags = map[model.TagName][]string{"album": {"Album Name"}}
				Expect(getPID(mf, md, spec)).To(Equal("(album name)"))
			})
		})
	})
})
