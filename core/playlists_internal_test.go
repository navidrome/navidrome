package core

import (
	"context"

	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"golang.org/x/text/unicode/norm"
)

var _ = Describe("compileLibraryPaths", func() {
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

			libRegex, err := ps.compileLibraryPaths(ctx)
			Expect(err).ToNot(HaveOccurred())

			// Test that longest path matches first
			// Note: The regex pattern ^path(?:/|$) will match the path plus the trailing /
			testCases := []struct {
				path     string
				expected string
			}{
				{"/music-classical/opera/track.mp3", "/music-classical/opera/"},
				{"/music-classical/track.mp3", "/music-classical/"},
				{"/music/track.mp3", "/music/"},
				{"/music-classical/opera/", "/music-classical/opera/"}, // Trailing slash
				{"/music-classical/opera", "/music-classical/opera"},   // Exact match (no trailing /)
			}

			for _, tc := range testCases {
				matched := libRegex.FindString(tc.path)
				Expect(matched).To(Equal(tc.expected), "Path %s should match %s, but got %s", tc.path, tc.expected, matched)
			}
		})

		It("handles libraries with similar prefixes but different structures", func() {
			mockLibRepo.SetData([]model.Library{
				{ID: 1, Path: "/home/user/music"},
				{ID: 2, Path: "/home/user/music-backup"},
			})

			libRegex, err := ps.compileLibraryPaths(ctx)
			Expect(err).ToNot(HaveOccurred())

			// Test that music-backup library is matched correctly
			matched := libRegex.FindString("/home/user/music-backup/track.mp3")
			Expect(matched).To(Equal("/home/user/music-backup/"))

			// Test that music library is still matched correctly
			matched = libRegex.FindString("/home/user/music/track.mp3")
			Expect(matched).To(Equal("/home/user/music/"))
		})

		It("matches path that is exactly the library root", func() {
			mockLibRepo.SetData([]model.Library{
				{ID: 1, Path: "/music"},
				{ID: 2, Path: "/music-classical"},
			})

			libRegex, err := ps.compileLibraryPaths(ctx)
			Expect(err).ToNot(HaveOccurred())

			// Exact library path should match (no trailing /)
			matched := libRegex.FindString("/music-classical")
			Expect(matched).To(Equal("/music-classical"))
		})

		It("handles complex nested library structures", func() {
			mockLibRepo.SetData([]model.Library{
				{ID: 1, Path: "/media"},
				{ID: 2, Path: "/media/audio"},
				{ID: 3, Path: "/media/audio/classical"},
				{ID: 4, Path: "/media/audio/classical/baroque"},
			})

			libRegex, err := ps.compileLibraryPaths(ctx)
			Expect(err).ToNot(HaveOccurred())

			testCases := []struct {
				path     string
				expected string
			}{
				{"/media/audio/classical/baroque/bach/track.mp3", "/media/audio/classical/baroque/"},
				{"/media/audio/classical/mozart/track.mp3", "/media/audio/classical/"},
				{"/media/audio/rock/track.mp3", "/media/audio/"},
				{"/media/video/movie.mp4", "/media/"},
			}

			for _, tc := range testCases {
				matched := libRegex.FindString(tc.path)
				Expect(matched).To(Equal(tc.expected), "Path %s should match %s, but got %s", tc.path, tc.expected, matched)
			}
		})
	})

	Describe("Edge cases", func() {
		It("handles empty library list", func() {
			mockLibRepo.SetData([]model.Library{})

			libRegex, err := ps.compileLibraryPaths(ctx)
			Expect(err).ToNot(HaveOccurred())
			Expect(libRegex).ToNot(BeNil())

			// Should not match anything
			matched := libRegex.FindString("/music/track.mp3")
			Expect(matched).To(BeEmpty())
		})

		It("handles single library", func() {
			mockLibRepo.SetData([]model.Library{
				{ID: 1, Path: "/music"},
			})

			libRegex, err := ps.compileLibraryPaths(ctx)
			Expect(err).ToNot(HaveOccurred())

			matched := libRegex.FindString("/music/track.mp3")
			Expect(matched).To(Equal("/music/"))
		})

		It("handles libraries with special regex characters", func() {
			mockLibRepo.SetData([]model.Library{
				{ID: 1, Path: "/music[test]"},
				{ID: 2, Path: "/music(backup)"},
			})

			libRegex, err := ps.compileLibraryPaths(ctx)
			Expect(err).ToNot(HaveOccurred())
			Expect(libRegex).ToNot(BeNil())

			// Special characters should be escaped and match literally
			matched := libRegex.FindString("/music[test]/track.mp3")
			Expect(matched).To(Equal("/music[test]/"))
		})
	})

	Describe("Regex pattern validation", func() {
		It("ensures regex alternation respects order by testing actual matching behavior", func() {
			mockLibRepo.SetData([]model.Library{
				{ID: 1, Path: "/a"},
				{ID: 2, Path: "/ab"},
				{ID: 3, Path: "/abc"},
			})

			libRegex, err := ps.compileLibraryPaths(ctx)
			Expect(err).ToNot(HaveOccurred())

			// Verify that longer paths match correctly (not cut off by shorter prefix)
			// If ordering is wrong, /ab would match before /abc for path "/abc/file"
			testCases := []struct {
				path     string
				expected string
			}{
				{"/abc/file.mp3", "/abc/"},
				{"/ab/file.mp3", "/ab/"},
				{"/a/file.mp3", "/a/"},
			}

			for _, tc := range testCases {
				matched := libRegex.FindString(tc.path)
				Expect(matched).To(Equal(tc.expected), "Path %s should match %s", tc.path, tc.expected)
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
