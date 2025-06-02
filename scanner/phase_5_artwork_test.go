package scanner

import (
	"context"
	"os"
	"path/filepath"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/conf/configtest"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("phaseArtwork", func() {
	var ctx context.Context
	var ds model.DataStore
	var tempDir string
	var phase *phaseArtwork

	BeforeEach(func() {
		DeferCleanup(configtest.SetupConfig())
		ctx = context.Background()
		ds = &tests.MockDataStore{}

		// Create temporary artwork directory
		var err error
		tempDir, err = os.MkdirTemp("", "navidrome-artwork-test")
		Expect(err).ToNot(HaveOccurred())

		conf.Server.ArtworkFolder = tempDir

		// Create playlist artwork subdirectory
		playlistDir := filepath.Join(tempDir, "playlist")
		err = os.MkdirAll(playlistDir, 0755)
		Expect(err).ToNot(HaveOccurred())

		state := &scanState{}
		phase = createPhaseArtwork(ctx, state, ds)

		DeferCleanup(func() {
			os.RemoveAll(tempDir)
		})
	})

	Describe("calculateFileHash", func() {
		It("should calculate hash including file content and modification time", func() {
			// Create test image file
			testFile := filepath.Join(tempDir, "playlist", "test-playlist.jpg")
			err := os.WriteFile(testFile, []byte("fake image content"), 0600)
			Expect(err).ToNot(HaveOccurred())

			hash1, err := phase.calculateFileHash(testFile)
			Expect(err).ToNot(HaveOccurred())
			Expect(hash1).ToNot(BeEmpty())
			Expect(len(hash1)).To(Equal(8)) // First 8 chars of MD5

			// Modify file content
			err = os.WriteFile(testFile, []byte("different image content"), 0600)
			Expect(err).ToNot(HaveOccurred())

			hash2, err := phase.calculateFileHash(testFile)
			Expect(err).ToNot(HaveOccurred())
			Expect(hash2).ToNot(Equal(hash1)) // Hash should change
		})
	})

	Describe("isValidImageFile", func() {
		DescribeTable("image file validation",
			func(filename string, expected bool) {
				Expect(isValidImageFile(filename)).To(Equal(expected))
			},
			Entry("JPEG file", "playlist.jpg", true),
			Entry("PNG file", "playlist.png", true),
			Entry("GIF file", "playlist.gif", true),
			Entry("text file", "playlist.txt", false),
			Entry("no extension", "playlist", false),
			Entry("case insensitive", "playlist.JPG", true),
		)
	})

	Describe("produce", func() {
		It("should find image files in playlist folder", func() {
			// Create test files
			validFiles := []string{"playlist1.jpg", "playlist2.png"}
			invalidFiles := []string{"readme.txt", "playlist3"}

			for _, file := range validFiles {
				path := filepath.Join(tempDir, "playlist", file)
				err := os.WriteFile(path, []byte("test"), 0600)
				Expect(err).ToNot(HaveOccurred())
			}

			for _, file := range invalidFiles {
				path := filepath.Join(tempDir, "playlist", file)
				err := os.WriteFile(path, []byte("test"), 0600)
				Expect(err).ToNot(HaveOccurred())
			}

			var foundFiles []string
			err := phase.produce(func(file string) {
				foundFiles = append(foundFiles, filepath.Base(file))
			})

			Expect(err).ToNot(HaveOccurred())
			Expect(foundFiles).To(HaveLen(2))
			Expect(foundFiles).To(ContainElements("playlist1.jpg", "playlist2.png"))
			Expect(foundFiles).ToNot(ContainElements("readme.txt", "playlist3"))
		})

		It("should handle missing artwork folder gracefully", func() {
			conf.Server.ArtworkFolder = ""

			var foundFiles []string
			err := phase.produce(func(file string) {
				foundFiles = append(foundFiles, file)
			})

			Expect(err).ToNot(HaveOccurred())
			Expect(foundFiles).To(BeEmpty())
		})
	})
})
