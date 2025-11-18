package core

import (
	"context"

	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("libraryMatcher", func() {
	var ds *tests.MockDataStore
	var mockLibRepo *tests.MockLibraryRepo
	ctx := context.Background()

	BeforeEach(func() {
		mockLibRepo = &tests.MockLibraryRepo{}
		ds = &tests.MockDataStore{
			MockedLibrary: mockLibRepo,
		}
	})

	// Helper function to create a libraryMatcher from the mock datastore
	createMatcher := func(ds model.DataStore) *libraryMatcher {
		libs, err := ds.Library(ctx).GetAll()
		Expect(err).ToNot(HaveOccurred())
		return newLibraryMatcher(libs)
	}

	Describe("Longest library path matching", func() {
		It("matches the longest library path when multiple libraries share a prefix", func() {
			// Setup libraries with prefix conflicts
			mockLibRepo.SetData([]model.Library{
				{ID: 1, Path: "/music"},
				{ID: 2, Path: "/music-classical"},
				{ID: 3, Path: "/music-classical/opera"},
			})

			matcher := createMatcher(ds)

			// Test that longest path matches first and returns correct library ID
			testCases := []struct {
				path            string
				expectedLibID   int
				expectedLibPath string
			}{
				{"/music-classical/opera/track.mp3", 3, "/music-classical/opera"},
				{"/music-classical/track.mp3", 2, "/music-classical"},
				{"/music/track.mp3", 1, "/music"},
				{"/music-classical/opera/subdir/file.mp3", 3, "/music-classical/opera"},
			}

			for _, tc := range testCases {
				libID, libPath := matcher.findLibraryForPath(tc.path)
				Expect(libID).To(Equal(tc.expectedLibID), "Path %s should match library ID %d, but got %d", tc.path, tc.expectedLibID, libID)
				Expect(libPath).To(Equal(tc.expectedLibPath), "Path %s should match library path %s, but got %s", tc.path, tc.expectedLibPath, libPath)
			}
		})

		It("handles libraries with similar prefixes but different structures", func() {
			mockLibRepo.SetData([]model.Library{
				{ID: 1, Path: "/home/user/music"},
				{ID: 2, Path: "/home/user/music-backup"},
			})

			matcher := createMatcher(ds)

			// Test that music-backup library is matched correctly
			libID, libPath := matcher.findLibraryForPath("/home/user/music-backup/track.mp3")
			Expect(libID).To(Equal(2))
			Expect(libPath).To(Equal("/home/user/music-backup"))

			// Test that music library is still matched correctly
			libID, libPath = matcher.findLibraryForPath("/home/user/music/track.mp3")
			Expect(libID).To(Equal(1))
			Expect(libPath).To(Equal("/home/user/music"))
		})

		It("matches path that is exactly the library root", func() {
			mockLibRepo.SetData([]model.Library{
				{ID: 1, Path: "/music"},
				{ID: 2, Path: "/music-classical"},
			})

			matcher := createMatcher(ds)

			// Exact library path should match
			libID, libPath := matcher.findLibraryForPath("/music-classical")
			Expect(libID).To(Equal(2))
			Expect(libPath).To(Equal("/music-classical"))
		})

		It("handles complex nested library structures", func() {
			mockLibRepo.SetData([]model.Library{
				{ID: 1, Path: "/media"},
				{ID: 2, Path: "/media/audio"},
				{ID: 3, Path: "/media/audio/classical"},
				{ID: 4, Path: "/media/audio/classical/baroque"},
			})

			matcher := createMatcher(ds)

			testCases := []struct {
				path            string
				expectedLibID   int
				expectedLibPath string
			}{
				{"/media/audio/classical/baroque/bach/track.mp3", 4, "/media/audio/classical/baroque"},
				{"/media/audio/classical/mozart/track.mp3", 3, "/media/audio/classical"},
				{"/media/audio/rock/track.mp3", 2, "/media/audio"},
				{"/media/video/movie.mp4", 1, "/media"},
			}

			for _, tc := range testCases {
				libID, libPath := matcher.findLibraryForPath(tc.path)
				Expect(libID).To(Equal(tc.expectedLibID), "Path %s should match library ID %d", tc.path, tc.expectedLibID)
				Expect(libPath).To(Equal(tc.expectedLibPath), "Path %s should match library path %s", tc.path, tc.expectedLibPath)
			}
		})
	})

	Describe("Edge cases", func() {
		It("handles empty library list", func() {
			mockLibRepo.SetData([]model.Library{})

			matcher := createMatcher(ds)
			Expect(matcher).ToNot(BeNil())

			// Should not match anything
			libID, libPath := matcher.findLibraryForPath("/music/track.mp3")
			Expect(libID).To(Equal(0))
			Expect(libPath).To(BeEmpty())
		})

		It("handles single library", func() {
			mockLibRepo.SetData([]model.Library{
				{ID: 1, Path: "/music"},
			})

			matcher := createMatcher(ds)

			libID, libPath := matcher.findLibraryForPath("/music/track.mp3")
			Expect(libID).To(Equal(1))
			Expect(libPath).To(Equal("/music"))
		})

		It("handles libraries with special characters in paths", func() {
			mockLibRepo.SetData([]model.Library{
				{ID: 1, Path: "/music[test]"},
				{ID: 2, Path: "/music(backup)"},
			})

			matcher := createMatcher(ds)
			Expect(matcher).ToNot(BeNil())

			// Special characters should match literally
			libID, libPath := matcher.findLibraryForPath("/music[test]/track.mp3")
			Expect(libID).To(Equal(1))
			Expect(libPath).To(Equal("/music[test]"))
		})
	})

	Describe("Path matching order", func() {
		It("ensures longest paths match first", func() {
			mockLibRepo.SetData([]model.Library{
				{ID: 1, Path: "/a"},
				{ID: 2, Path: "/ab"},
				{ID: 3, Path: "/abc"},
			})

			matcher := createMatcher(ds)

			// Verify that longer paths match correctly (not cut off by shorter prefix)
			testCases := []struct {
				path          string
				expectedLibID int
			}{
				{"/abc/file.mp3", 3},
				{"/ab/file.mp3", 2},
				{"/a/file.mp3", 1},
			}

			for _, tc := range testCases {
				libID, _ := matcher.findLibraryForPath(tc.path)
				Expect(libID).To(Equal(tc.expectedLibID), "Path %s should match library ID %d", tc.path, tc.expectedLibID)
			}
		})
	})
})

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
		It("resolves absolute paths", func() {
			resolution := resolver.resolvePath("/music/artist/album/track.mp3", nil)

			Expect(resolution.valid).To(BeTrue())
			Expect(resolution.libraryID).To(Equal(1))
			Expect(resolution.libraryPath).To(Equal("/music"))
			Expect(resolution.absolutePath).To(Equal("/music/artist/album/track.mp3"))
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
			Expect(resolution.absolutePath).To(Equal("/music/artist/album/track.mp3"))
		})

		It("returns invalid resolution for paths outside any library", func() {
			resolution := resolver.resolvePath("/outside/library/track.mp3", nil)

			Expect(resolution.valid).To(BeFalse())
		})
	})

	Describe("resolvePath", func() {
		Context("With absolute paths", func() {
			It("resolves path within a library", func() {
				resolution := resolver.resolvePath("/music/track.mp3", nil)

				Expect(resolution.valid).To(BeTrue())
				Expect(resolution.libraryID).To(Equal(1))
				Expect(resolution.libraryPath).To(Equal("/music"))
				Expect(resolution.absolutePath).To(Equal("/music/track.mp3"))
			})

			It("resolves path to the longest matching library", func() {
				resolution := resolver.resolvePath("/music-classical/track.mp3", nil)

				Expect(resolution.valid).To(BeTrue())
				Expect(resolution.libraryID).To(Equal(2))
				Expect(resolution.libraryPath).To(Equal("/music-classical"))
			})

			It("returns invalid resolution for path outside libraries", func() {
				resolution := resolver.resolvePath("/videos/movie.mp4", nil)

				Expect(resolution.valid).To(BeFalse())
			})

			It("cleans the path before matching", func() {
				resolution := resolver.resolvePath("/music//artist/../artist/track.mp3", nil)

				Expect(resolution.valid).To(BeTrue())
				Expect(resolution.absolutePath).To(Equal("/music/artist/track.mp3"))
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
				Expect(resolution.absolutePath).To(Equal("/music/songs/track.mp3"))
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
				Expect(resolution.libraryPath).To(Equal("/podcasts"))
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
				Expect(resolution.libraryPath).To(Equal("/music-classical"))
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
			Expect(resolution.libraryPath).To(Equal("/podcasts"))
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
