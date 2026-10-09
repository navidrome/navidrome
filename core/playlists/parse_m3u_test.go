package playlists

import (
	"context"
	"path/filepath"

	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("pathResolver", func() {
	var ds *tests.MockDataStore
	var mockLibRepo *tests.MockLibraryRepo
	var resolver *pathResolver
	ctx := context.Background()

	BeforeEach(func() {
		mockLibRepo = &tests.MockLibraryRepo{}
		ds = &tests.MockDataStore{
			MockedLibrary: mockLibRepo,
		}

		// Setup test libraries
		mockLibRepo.SetData([]model.Library{
			{ID: 1, Path: "/music"},
			{ID: 2, Path: "/music-classical"},
			{ID: 3, Path: "/podcasts"},
		})

		var err error
		resolver, err = newPathResolver(ctx, ds)
		Expect(err).ToNot(HaveOccurred())
	})

	Describe("resolvePath", func() {
		Context("basic", func() {
			It("resolves absolute paths", func() {
				resolution := resolver.resolvePath("/music/artist/album/track.mp3", nil)

				Expect(resolution.valid).To(BeTrue())
				Expect(resolution.libraryID).To(Equal(1))
				Expect(resolution.libraryPath).To(Equal(filepath.FromSlash("/music")))
				Expect(resolution.absolutePath).To(Equal(filepath.FromSlash("/music/artist/album/track.mp3")))
			})

			It("resolves relative paths when folder is provided", func() {
				folder := &model.Folder{
					Path:        "playlists",
					LibraryPath: "/music",
					LibraryID:   1,
				}

				resolution := resolver.resolvePath("../artist/album/track.mp3", folder)

				Expect(resolution.valid).To(BeTrue())
				Expect(resolution.libraryID).To(Equal(1))
				Expect(resolution.absolutePath).To(Equal(filepath.FromSlash("/music/artist/album/track.mp3")))
			})

			It("returns invalid resolution for paths outside any library", func() {
				resolution := resolver.resolvePath("/outside/library/track.mp3", nil)

				Expect(resolution.valid).To(BeFalse())
			})
		})

		Context("cross-library", func() {
			It("resolves path within a library", func() {
				resolution := resolver.resolvePath("/music/track.mp3", nil)

				Expect(resolution.valid).To(BeTrue())
				Expect(resolution.libraryID).To(Equal(1))
				Expect(resolution.libraryPath).To(Equal(filepath.FromSlash("/music")))
				Expect(resolution.absolutePath).To(Equal(filepath.FromSlash("/music/track.mp3")))
			})

			It("resolves path to the longest matching library", func() {
				resolution := resolver.resolvePath("/music-classical/track.mp3", nil)

				Expect(resolution.valid).To(BeTrue())
				Expect(resolution.libraryID).To(Equal(2))
				Expect(resolution.libraryPath).To(Equal(filepath.FromSlash("/music-classical")))
			})

			It("returns invalid resolution for path outside libraries", func() {
				resolution := resolver.resolvePath("/videos/movie.mp4", nil)

				Expect(resolution.valid).To(BeFalse())
			})

			It("cleans the path before matching", func() {
				resolution := resolver.resolvePath("/music//artist/../artist/track.mp3", nil)

				Expect(resolution.valid).To(BeTrue())
				Expect(resolution.absolutePath).To(Equal(filepath.FromSlash("/music/artist/track.mp3")))
			})
		})

		Context("With relative paths", func() {
			It("resolves relative path within same library", func() {
				folder := &model.Folder{
					Path:        "playlists",
					LibraryPath: "/music",
					LibraryID:   1,
				}

				resolution := resolver.resolvePath("../songs/track.mp3", folder)

				Expect(resolution.valid).To(BeTrue())
				Expect(resolution.libraryID).To(Equal(1))
				Expect(resolution.absolutePath).To(Equal(filepath.FromSlash("/music/songs/track.mp3")))
			})

			It("resolves relative path to different library", func() {
				folder := &model.Folder{
					Path:        "playlists",
					LibraryPath: "/music",
					LibraryID:   1,
				}

				// Path goes up and into a different library
				resolution := resolver.resolvePath("../../podcasts/episode.mp3", folder)

				Expect(resolution.valid).To(BeTrue())
				Expect(resolution.libraryID).To(Equal(3))
				Expect(resolution.libraryPath).To(Equal(filepath.FromSlash("/podcasts")))
			})

			It("uses matcher to find correct library for resolved path", func() {
				folder := &model.Folder{
					Path:        "playlists",
					LibraryPath: "/music",
					LibraryID:   1,
				}

				// This relative path resolves to music-classical library
				resolution := resolver.resolvePath("../../music-classical/track.mp3", folder)

				Expect(resolution.valid).To(BeTrue())
				Expect(resolution.libraryID).To(Equal(2))
				Expect(resolution.libraryPath).To(Equal(filepath.FromSlash("/music-classical")))
			})

			It("returns invalid for relative paths escaping all libraries", func() {
				folder := &model.Folder{
					Path:        "playlists",
					LibraryPath: "/music",
					LibraryID:   1,
				}

				resolution := resolver.resolvePath("../../../../etc/passwd", folder)

				Expect(resolution.valid).To(BeFalse())
			})
		})
	})

	Describe("Cross-library resolution scenarios", func() {
		It("handles playlist in library A referencing file in library B", func() {
			// Playlist is in /music/playlists
			folder := &model.Folder{
				Path:        "playlists",
				LibraryPath: "/music",
				LibraryID:   1,
			}

			// Relative path that goes to /podcasts library
			resolution := resolver.resolvePath("../../podcasts/show/episode.mp3", folder)

			Expect(resolution.valid).To(BeTrue())
			Expect(resolution.libraryID).To(Equal(3), "Should resolve to podcasts library")
			Expect(resolution.libraryPath).To(Equal(filepath.FromSlash("/podcasts")))
		})

		It("prefers longer library paths when resolving", func() {
			// Ensure /music-classical is matched instead of /music
			resolution := resolver.resolvePath("/music-classical/baroque/track.mp3", nil)

			Expect(resolution.valid).To(BeTrue())
			Expect(resolution.libraryID).To(Equal(2), "Should match /music-classical, not /music")
		})
	})
})

var _ = Describe("pathResolution", func() {
	Describe("ToQualifiedString", func() {
		It("converts valid resolution to qualified string with forward slashes", func() {
			resolution := pathResolution{
				absolutePath: "/music/artist/album/track.mp3",
				libraryPath:  "/music",
				libraryID:    1,
				valid:        true,
			}

			qualifiedStr, err := resolution.ToQualifiedString()

			Expect(err).ToNot(HaveOccurred())
			Expect(qualifiedStr).To(Equal("1:artist/album/track.mp3"))
		})

		It("handles Windows-style paths by converting to forward slashes", func() {
			resolution := pathResolution{
				absolutePath: "/music/artist/album/track.mp3",
				libraryPath:  "/music",
				libraryID:    2,
				valid:        true,
			}

			qualifiedStr, err := resolution.ToQualifiedString()

			Expect(err).ToNot(HaveOccurred())
			// Should always use forward slashes regardless of OS
			Expect(qualifiedStr).To(ContainSubstring("2:"))
			Expect(qualifiedStr).ToNot(ContainSubstring("\\"))
		})

		It("returns error for invalid resolution", func() {
			resolution := pathResolution{valid: false}

			_, err := resolution.ToQualifiedString()

			Expect(err).To(HaveOccurred())
		})
	})
})
