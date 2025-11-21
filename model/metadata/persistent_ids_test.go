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
		getPID getPIDFunc
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
				pid := getPID(mf, md, spec, false)
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
				Expect(getPID(mf, md, spec, false)).To(Equal("(mbtrackid)"))
			})
		})
		When("only first field is present", func() {
			It("should return the pid", func() {
				md.tags = map[model.TagName][]string{
					"musicbrainz_trackid": {"mbtrackid"},
				}
				Expect(getPID(mf, md, spec, false)).To(Equal("(mbtrackid)"))
			})
		})
		When("first is empty, but second field is present", func() {
			It("should return the pid", func() {
				md.tags = map[model.TagName][]string{
					"album":      {"album name"},
					"discnumber": {"1"},
				}
				Expect(getPID(mf, md, spec, false)).To(Equal("(album name\\1\\)"))
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
				Expect(getPID(mf, md, spec, false)).To(Equal("(Title)"))
			})
		})
		When("field is folder", func() {
			It("should return the pid", func() {
				spec := "folder|title"
				md.tags = map[model.TagName][]string{"title": {"title"}}
				mf.Path = "/path/to/file.mp3"
				Expect(getPID(mf, md, spec, false)).To(Equal("(/path/to)"))
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
				Expect(getPID(mf, md, spec, false)).To(Equal("(((album artist)\\album name\\version\\2021-01-01))"))
			})
		})
		When("field is albumartistid", func() {
			It("should return the pid", func() {
				spec := "musicbrainz_albumartistid|albumartistid"
				md.tags = map[model.TagName][]string{
					"albumartist": {"Album Artist"},
				}
				mf.AlbumArtist = "Album Artist"
				Expect(getPID(mf, md, spec, false)).To(Equal("((album artist))"))
			})
		})
		When("field is album", func() {
			It("should return the pid", func() {
				spec := "album|title"
				md.tags = map[model.TagName][]string{"album": {"Album Name"}}
				Expect(getPID(mf, md, spec, false)).To(Equal("(album name)"))
			})
		})
	})

	Context("edge cases", func() {
		When("the spec has spaces between groups", func() {
			It("should return the pid", func() {
				spec := "albumartist| Album"
				md.tags = map[model.TagName][]string{
					"album": {"album name"},
				}
				Expect(getPID(mf, md, spec, false)).To(Equal("(album name)"))
			})
		})
		When("the spec has spaces", func() {
			It("should return the pid", func() {
				spec := "albumartist, album"
				md.tags = map[model.TagName][]string{
					"albumartist": {"Album Artist"},
					"album":       {"album name"},
				}
				Expect(getPID(mf, md, spec, false)).To(Equal("(Album Artist\\album name)"))
			})
		})
		When("the spec has mixed case fields", func() {
			It("should return the pid", func() {
				spec := "albumartist,Album"
				md.tags = map[model.TagName][]string{
					"albumartist": {"Album Artist"},
					"album":       {"album name"},
				}
				Expect(getPID(mf, md, spec, false)).To(Equal("(Album Artist\\album name)"))
			})
		})
	})

	Context("prependLibId functionality", func() {
		BeforeEach(func() {
			mf.LibraryID = 42
		})
		When("prependLibId is true", func() {
			It("should prepend library ID to the hash input", func() {
				spec := "album"
				md.tags = map[model.TagName][]string{"album": {"Test Album"}}
				pid := getPID(mf, md, spec, true)
				// The hash function should receive "42\test album" as input
				Expect(pid).To(Equal("(42\\test album)"))
			})
		})
		When("prependLibId is false", func() {
			It("should not prepend library ID to the hash input", func() {
				spec := "album"
				md.tags = map[model.TagName][]string{"album": {"Test Album"}}
				pid := getPID(mf, md, spec, false)
				// The hash function should receive "test album" as input
				Expect(pid).To(Equal("(test album)"))
			})
		})
		When("prependLibId is true with complex spec", func() {
			It("should prepend library ID to the final hash input", func() {
				spec := "musicbrainz_trackid|album,tracknumber"
				md.tags = map[model.TagName][]string{
					"album":       {"Test Album"},
					"tracknumber": {"1"},
				}
				pid := getPID(mf, md, spec, true)
				// Should use the fallback field and prepend library ID
				Expect(pid).To(Equal("(42\\test album\\1)"))
			})
		})
		When("prependLibId is true with nested albumid", func() {
			It("should handle nested albumid calls correctly", func() {
				DeferCleanup(configtest.SetupConfig())
				conf.Server.PID.Album = "album"
				spec := "albumid"
				md.tags = map[model.TagName][]string{"album": {"Test Album"}}
				mf.AlbumArtist = "Test Artist"
				pid := getPID(mf, md, spec, true)
				// The albumid call should also use prependLibId=true
				Expect(pid).To(Equal("(42\\(42\\test album))"))
			})
		})
	})

	Context("legacy specs", func() {
		Context("track_legacy", func() {
			When("library ID is default (1)", func() {
				It("should not prepend library ID even when prependLibId is true", func() {
					mf.Path = "/path/to/track.mp3"
					mf.LibraryID = 1 // Default library ID
					// With default library, both should be the same
					pidTrue := getPID(mf, md, "track_legacy", true)
					pidFalse := getPID(mf, md, "track_legacy", false)
					Expect(pidTrue).To(Equal(pidFalse))
					Expect(pidTrue).NotTo(BeEmpty())
				})
			})
			When("library ID is non-default", func() {
				It("should prepend library ID when prependLibId is true", func() {
					mf.Path = "/path/to/track.mp3"
					mf.LibraryID = 2 // Non-default library ID
					pidTrue := getPID(mf, md, "track_legacy", true)
					pidFalse := getPID(mf, md, "track_legacy", false)
					Expect(pidTrue).NotTo(Equal(pidFalse))
					Expect(pidTrue).NotTo(BeEmpty())
					Expect(pidFalse).NotTo(BeEmpty())
				})
			})
			When("library ID is non-default but prependLibId is false", func() {
				It("should not prepend library ID", func() {
					mf.Path = "/path/to/track.mp3"
					mf.LibraryID = 3
					mf2 := mf
					mf2.LibraryID = 1 // Default library
					pidNonDefault := getPID(mf, md, "track_legacy", false)
					pidDefault := getPID(mf2, md, "track_legacy", false)
					// Should be the same since prependLibId=false
					Expect(pidNonDefault).To(Equal(pidDefault))
				})
			})
		})
		Context("album_legacy", func() {
			When("library ID is default (1)", func() {
				It("should not prepend library ID even when prependLibId is true", func() {
					md.tags = map[model.TagName][]string{"album": {"Test Album"}}
					mf.LibraryID = 1 // Default library ID
					pidTrue := getPID(mf, md, "album_legacy", true)
					pidFalse := getPID(mf, md, "album_legacy", false)
					Expect(pidTrue).To(Equal(pidFalse))
					Expect(pidTrue).NotTo(BeEmpty())
				})
			})
			When("library ID is non-default", func() {
				It("should prepend library ID when prependLibId is true", func() {
					md.tags = map[model.TagName][]string{"album": {"Test Album"}}
					mf.LibraryID = 2 // Non-default library ID
					pidTrue := getPID(mf, md, "album_legacy", true)
					pidFalse := getPID(mf, md, "album_legacy", false)
					Expect(pidTrue).NotTo(Equal(pidFalse))
					Expect(pidTrue).NotTo(BeEmpty())
					Expect(pidFalse).NotTo(BeEmpty())
				})
			})
			When("library ID is non-default but prependLibId is false", func() {
				It("should not prepend library ID", func() {
					md.tags = map[model.TagName][]string{"album": {"Test Album"}}
					mf.LibraryID = 3
					mf2 := mf
					mf2.LibraryID = 1 // Default library
					pidNonDefault := getPID(mf, md, "album_legacy", false)
					pidDefault := getPID(mf2, md, "album_legacy", false)
					// Should be the same since prependLibId=false
					Expect(pidNonDefault).To(Equal(pidDefault))
				})
			})
		})
	})
})
