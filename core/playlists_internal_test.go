package core

import (
	"context"

	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"golang.org/x/text/unicode/norm"
)

var _ = Describe("buildLibraryMatcher", func() {
	var ds *tests.MockDataStore
	var mockLibRepo *tests.MockLibraryRepo
	var ps *playlists
	ctx := context.Background()

	BeforeEach(func() {
		mockLibRepo = &tests.MockLibraryRepo{}
		ds = &tests.MockDataStore{
			MockedLibrary: mockLibRepo,
		}
		ps = &playlists{ds: ds}
	})

	Describe("Longest library path matching", func() {
		It("matches the longest library path when multiple libraries share a prefix", func() {
			// Setup libraries with prefix conflicts
			mockLibRepo.SetData([]model.Library{
				{ID: 1, Path: "/music"},
				{ID: 2, Path: "/music-classical"},
				{ID: 3, Path: "/music-classical/opera"},
			})

			matcher, err := ps.buildLibraryMatcher(ctx)
			Expect(err).ToNot(HaveOccurred())

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

			matcher, err := ps.buildLibraryMatcher(ctx)
			Expect(err).ToNot(HaveOccurred())

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

			matcher, err := ps.buildLibraryMatcher(ctx)
			Expect(err).ToNot(HaveOccurred())

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

			matcher, err := ps.buildLibraryMatcher(ctx)
			Expect(err).ToNot(HaveOccurred())

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

			matcher, err := ps.buildLibraryMatcher(ctx)
			Expect(err).ToNot(HaveOccurred())
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

			matcher, err := ps.buildLibraryMatcher(ctx)
			Expect(err).ToNot(HaveOccurred())

			libID, libPath := matcher.findLibraryForPath("/music/track.mp3")
			Expect(libID).To(Equal(1))
			Expect(libPath).To(Equal("/music"))
		})

		It("handles libraries with special characters in paths", func() {
			mockLibRepo.SetData([]model.Library{
				{ID: 1, Path: "/music[test]"},
				{ID: 2, Path: "/music(backup)"},
			})

			matcher, err := ps.buildLibraryMatcher(ctx)
			Expect(err).ToNot(HaveOccurred())
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

			matcher, err := ps.buildLibraryMatcher(ctx)
			Expect(err).ToNot(HaveOccurred())

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

var _ = Describe("normalizePathForComparison", func() {
	It("normalizes Unicode characters to NFC form and converts to lowercase", func() {
		// Test with NFD (decomposed) input - as would come from macOS filesystem
		nfdPath := norm.NFD.String("Michèle") // Explicitly convert to NFD form
		normalized := normalizePathForComparison(nfdPath)
		Expect(normalized).To(Equal("michèle"))

		// Test with NFC (composed) input - as would come from Apple Music M3U
		nfcPath := "Michèle" // This might be in NFC form
		normalizedNfc := normalizePathForComparison(nfcPath)

		// Ensure the two paths are not equal in their original forms
		Expect(nfdPath).ToNot(Equal(nfcPath))

		// Both should normalize to the same result
		Expect(normalized).To(Equal(normalizedNfc))
	})

	It("handles paths with mixed case and Unicode characters", func() {
		path := "Artist/Noël Coward/Album/Song.mp3"
		normalized := normalizePathForComparison(path)
		Expect(normalized).To(Equal("artist/noël coward/album/song.mp3"))
	})
})
