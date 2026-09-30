package metadata

import (
	"strings"

	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("getPID", func() {
	var (
		md        Metadata
		mf        model.MediaFile
		sum       hashFunc
		albumSpec string
	)
	getPID := func(mf model.MediaFile, md Metadata, spec string, prependLibId bool) string {
		return computePID(mf, md, spec, albumSpec, prependLibId, sum)
	}

	BeforeEach(func() {
		sum = func(s ...string) string { return "(" + strings.Join(s, ",") + ")" }
		albumSpec = consts.DefaultAlbumPID
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
			albumSpec = "musicbrainz_albumid|albumartistid,album,albumversion,releasedate"
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
				tests.SkipOnWindows("path separator bug (#TBD-path-sep-metadata)")
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
					"title":        {"title"},
					"album":        {"album name"},
					"albumversion": {"deluxe edition"},
					"releasedate":  {"2021-01-01"},
				}
				mf.AlbumArtist = "Album Artist"
				Expect(getPID(mf, md, spec, false)).To(Equal("(((album artist)\\album name\\deluxe edition\\2021-01-01))"))
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

		When("albumid configuration refers to albumid recursively", func() {
			It("should avoid infinite recursion", func() {
				// Reproduce the issue from #4920
				albumSpec = "albumid,album,albumversion,releasedate"
				spec := albumSpec
				md.tags = map[model.TagName][]string{
					"album":        {"Album Name"},
					"albumversion": {"Version"},
					"releasedate":  {"2022"},
				}
				// Should not panic and return a valid PID ignoring the recursive "albumid"
				Expect(func() {
					pid := getPID(mf, md, spec, false)
					Expect(pid).To(Equal("(\\album name\\Version\\2022)"))
				}).To(Not(Panic()))
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
				albumSpec = "album"
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
		It("emits canonical 22-char base62 ids", func() {
			mf := model.MediaFile{Path: "/music/a.mp3", LibraryID: 1}
			// md5("/music/a.mp3") = e3b7fc2ae9447bbec37a13bf916e3cf6 re-encoded as base62
			Expect(legacyTrackID(mf, false)).To(Equal("6VHl3uR4kss6sUPKA8Cwnk"))
		})
		It("prepends the library id for a non-default library", func() {
			mf := model.MediaFile{Path: "/music/a.mp3", LibraryID: 2}
			// id.Encode(md5.Sum([]byte("2\\/music/a.mp3")))
			Expect(legacyTrackID(mf, true)).To(Equal("4EK5DHQBMeFuDHw6S3iooO"))
		})
		It("emits a canonical album id (golden)", func() {
			mf := model.MediaFile{LibraryID: 1}
			// id.Encode(md5.Sum([]byte("[unknown artist]\\[unknown album]")))
			Expect(legacyAlbumID(mf, Metadata{}, false)).To(Equal("6xBmxSAUFJSQuW7UvwCq8X"))
		})
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

var _ = Describe("ValidatePIDSpec", func() {
	DescribeTable("accepts valid specs",
		func(spec string, isAlbum bool) {
			Expect(ValidatePIDSpec(spec, isAlbum)).To(Succeed())
		},
		Entry("empty, meaning the global config", "", true),
		Entry("default album spec", consts.DefaultAlbumPID, true),
		Entry("default track spec, which uses tag aliases", consts.DefaultTrackPID, false),
		Entry("folder", "folder", true),
		Entry("album legacy", "album_legacy", true),
		Entry("track legacy", "track_legacy", false),
		Entry("computed attributes", "albumartistid,album|title", true),
		Entry("albumid in a track spec", "albumid,title", false),
		Entry("spaces and mixed case", "MusicBrainz_AlbumID | Folder", true),
	)

	DescribeTable("rejects invalid specs",
		func(spec string, isAlbum bool, msg string) {
			Expect(ValidatePIDSpec(spec, isAlbum)).To(MatchError(ContainSubstring(msg)))
		},
		Entry("unknown tag", "albmversion", true, `unknown attribute "albmversion"`),
		Entry("empty field", "album||title", true, "empty attribute"),
		Entry("empty attribute", "album,,title", true, "empty attribute"),
		Entry("trailing separator", "album|", true, "empty attribute"),
		Entry("albumid in an album spec", "albumid,album", true, "albumid"),
		Entry("tag alias in an album spec", "talb", true, `use the tag name "album" instead of its alias "talb"`),
		Entry("track legacy in an album spec", "track_legacy", true, `unknown attribute "track_legacy"`),
		Entry("album legacy in a track spec", "album_legacy", false, `unknown attribute "album_legacy"`),
	)
})
