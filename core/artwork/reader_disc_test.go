package artwork

import (
	"context"
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Disc Artwork Reader", func() {
	Describe("extractDiscNumber", func() {
		DescribeTable("extracts disc number from filename based on glob pattern",
			func(pattern, filename string, expectedNum int, expectedOk bool) {
				num, ok := extractDiscNumber(pattern, filename)
				Expect(ok).To(Equal(expectedOk))
				if expectedOk {
					Expect(num).To(Equal(expectedNum))
				}
			},
			// Standard disc patterns
			Entry("disc1.jpg", "disc*.*", "disc1.jpg", 1, true),
			Entry("disc2.png", "disc*.*", "disc2.png", 2, true),
			Entry("disc01.jpg", "disc*.*", "disc01.jpg", 1, true),
			Entry("disc02.png", "disc*.*", "disc02.png", 2, true),
			Entry("disc10.jpg", "disc*.*", "disc10.jpg", 10, true),

			// CD patterns
			Entry("cd1.jpg", "cd*.*", "cd1.jpg", 1, true),
			Entry("cd02.png", "cd*.*", "cd02.png", 2, true),

			// No number in filename
			Entry("disc.jpg has no number", "disc*.*", "disc.jpg", 0, false),
			Entry("cd.jpg has no number", "cd*.*", "cd.jpg", 0, false),

			// Extra text after number
			Entry("disc2-bonus.jpg", "disc*.*", "disc2-bonus.jpg", 2, true),
			Entry("disc01_front.png", "disc*.*", "disc01_front.png", 1, true),

			// Case insensitive (filename already lowered by caller)
			Entry("Disc1.jpg lowered", "disc*.*", "disc1.jpg", 1, true),

			// Pattern doesn't match
			Entry("cover.jpg doesn't match disc*.*", "disc*.*", "cover.jpg", 0, false),

			// Pattern with no wildcard before dot
			Entry("front1.jpg with front*.*", "front*.*", "front1.jpg", 1, true),
		)
	})

	Describe("fromDiscExternalFile", func() {
		var (
			ctx    context.Context
			tmpDir string
		)

		BeforeEach(func() {
			ctx = context.Background()
			tmpDir = GinkgoT().TempDir()
		})

		createFile := func(path string) string {
			fullPath := filepath.Join(tmpDir, filepath.FromSlash(path))
			Expect(os.MkdirAll(filepath.Dir(fullPath), 0755)).To(Succeed())
			Expect(os.WriteFile(fullPath, []byte("image data"), 0600)).To(Succeed())
			return fullPath
		}

		It("matches file with disc number in single-folder album", func() {
			f1 := createFile("album/disc1.jpg")
			f2 := createFile("album/disc2.jpg")
			discFolders := map[string]bool{filepath.Join(tmpDir, "album"): true}

			sf := fromDiscExternalFile(ctx, []string{f1, f2}, "disc*.*", 1, discFolders, false)
			r, path, err := sf()
			Expect(err).ToNot(HaveOccurred())
			Expect(r).ToNot(BeNil())
			r.Close()
			Expect(path).To(Equal(f1))
		})

		It("skips file without number in single-folder album", func() {
			f1 := createFile("album/disc.jpg")
			discFolders := map[string]bool{filepath.Join(tmpDir, "album"): true}

			sf := fromDiscExternalFile(ctx, []string{f1}, "disc*.*", 1, discFolders, false)
			r, _, _ := sf()
			Expect(r).To(BeNil())
		})

		It("matches file without number in multi-folder album by folder", func() {
			f1 := createFile("album/cd1/disc.jpg")
			f2 := createFile("album/cd2/disc.jpg")
			discFolders := map[string]bool{filepath.Join(tmpDir, "album", "cd1"): true}

			sf := fromDiscExternalFile(ctx, []string{f1, f2}, "disc*.*", 1, discFolders, true)
			r, path, err := sf()
			Expect(err).ToNot(HaveOccurred())
			Expect(r).ToNot(BeNil())
			r.Close()
			Expect(path).To(Equal(f1))
		})

		It("prefers disc number over folder when number is present", func() {
			// disc2.jpg in cd1 folder should match disc 2, not disc 1
			f1 := createFile("album/cd1/disc2.jpg")
			discFolders := map[string]bool{filepath.Join(tmpDir, "album", "cd1"): true}

			sf := fromDiscExternalFile(ctx, []string{f1}, "disc*.*", 2, discFolders, true)
			r, path, err := sf()
			Expect(err).ToNot(HaveOccurred())
			Expect(r).ToNot(BeNil())
			r.Close()
			Expect(path).To(Equal(f1))
		})

		It("does not match disc2.jpg when looking for disc 1", func() {
			f1 := createFile("album/disc2.jpg")
			discFolders := map[string]bool{filepath.Join(tmpDir, "album"): true}

			sf := fromDiscExternalFile(ctx, []string{f1}, "disc*.*", 1, discFolders, false)
			r, _, _ := sf()
			Expect(r).To(BeNil())
		})
	})

	Describe("discArtworkReader", func() {
		Describe("fromDiscArtPriority", func() {
			var reader *discArtworkReader

			BeforeEach(func() {
				reader = &discArtworkReader{
					discNumber:    2,
					isMultiFolder: true,
					discFolders:   map[string]bool{"/music/album/cd2": true},
					imgFiles: []string{
						"/music/album/cd1/disc.jpg",
						"/music/album/cd2/disc.jpg",
						"/music/album/cd2/disc2.jpg",
					},
					firstTrackPath: "/music/album/cd2/track1.flac",
					rootFolder:     "/music",
				}
			})

			It("returns source funcs for glob patterns", func() {
				ff := reader.fromDiscArtPriority(context.Background(), nil, "disc*.*")
				Expect(ff).To(HaveLen(1))
			})

			It("returns source funcs for embedded pattern", func() {
				ff := reader.fromDiscArtPriority(context.Background(), nil, "embedded")
				Expect(ff).To(HaveLen(2)) // fromTag + fromFFmpegTag
			})

			It("handles multiple comma-separated patterns", func() {
				ff := reader.fromDiscArtPriority(context.Background(), nil, "disc*.*, cd*.*, embedded")
				Expect(ff).To(HaveLen(4)) // disc*.* + cd*.* + fromTag + fromFFmpegTag
			})

			It("ignores 'external' pattern silently", func() {
				ff := reader.fromDiscArtPriority(context.Background(), nil, "external")
				Expect(ff).To(HaveLen(0))
			})

			It("returns no source funcs when imgFiles is empty and pattern is not embedded", func() {
				reader.imgFiles = nil
				ff := reader.fromDiscArtPriority(context.Background(), nil, "disc*.*")
				Expect(ff).To(HaveLen(0))
			})
		})
	})
})
