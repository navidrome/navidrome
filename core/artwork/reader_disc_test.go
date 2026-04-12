package artwork

import (
	"context"
	"os"
	"path/filepath"

	"github.com/navidrome/navidrome/model"
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

			// HasPrefix guard: filename doesn't share the pattern's literal prefix
			Entry("cover.jpg with disc*.* (no prefix match)", "disc*.*", "cover.jpg", 0, false),

			// Pattern with no wildcard before dot
			Entry("front1.jpg with front*.*", "front*.*", "front1.jpg", 1, true),

			// '?' single-char wildcard
			Entry("disc?.jpg with disc1.jpg", "disc?.jpg", "disc1.jpg", 1, true),
			Entry("disc?.jpg with disc2.jpg", "disc?.jpg", "disc2.jpg", 2, true),
			Entry("cd??.jpg with cd07.jpg", "cd??.jpg", "cd07.jpg", 7, true),

			// '[...]' character class wildcard
			Entry("cd[12].jpg with cd1.jpg", "cd[12].jpg", "cd1.jpg", 1, true),
			Entry("cd[12].jpg with cd2.jpg", "cd[12].jpg", "cd2.jpg", 2, true),
			Entry("disc[0-9].jpg with disc5.jpg", "disc[0-9].jpg", "disc5.jpg", 5, true),

			// Literal pattern (no wildcard) returns false
			Entry("shellac.png literal", "shellac.png", "shellac.png", 0, false),
		)
	})

	Describe("fromExternalFile", func() {
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
			reader := &discArtworkReader{
				discNumber:  1,
				imgFiles:    []string{f1, f2},
				discFolders: map[string]bool{filepath.Join(tmpDir, "album"): true},
			}

			sf := reader.fromExternalFile(ctx, "disc*.*")
			r, path, err := sf()
			Expect(err).ToNot(HaveOccurred())
			Expect(r).ToNot(BeNil())
			r.Close()
			Expect(path).To(Equal(f1))
		})

		It("matches file without number in single-folder album (shared disc art)", func() {
			f1 := createFile("album/cover.png")
			reader := &discArtworkReader{
				discNumber:  1,
				imgFiles:    []string{f1},
				discFolders: map[string]bool{filepath.Join(tmpDir, "album"): true},
			}

			sf := reader.fromExternalFile(ctx, "cover.*")
			r, path, err := sf()
			Expect(err).ToNot(HaveOccurred())
			Expect(r).ToNot(BeNil())
			r.Close()
			Expect(path).To(Equal(f1))
		})

		It("returns shared disc art for every disc number in single-folder album", func() {
			f1 := createFile("album/shellac.png")
			makeReader := func(discNum int) *discArtworkReader {
				return &discArtworkReader{
					discNumber:  discNum,
					imgFiles:    []string{f1},
					discFolders: map[string]bool{filepath.Join(tmpDir, "album"): true},
				}
			}

			for _, disc := range []int{1, 2, 5} {
				sf := makeReader(disc).fromExternalFile(ctx, "shellac.png")
				r, path, err := sf()
				Expect(err).ToNot(HaveOccurred(), "disc %d", disc)
				Expect(r).ToNot(BeNil())
				r.Close()
				Expect(path).To(Equal(f1), "disc %d", disc)
			}
		})

		It("numbered and unnumbered patterns both resolve against the same reader", func() {
			f1 := createFile("album/cover.png")
			f2 := createFile("album/disc1.jpg")
			f3 := createFile("album/disc2.jpg")
			reader := &discArtworkReader{
				discNumber:  2,
				imgFiles:    []string{f1, f2, f3},
				discFolders: map[string]bool{filepath.Join(tmpDir, "album"): true},
			}

			sf := reader.fromExternalFile(ctx, "disc*.*")
			r, path, err := sf()
			Expect(err).ToNot(HaveOccurred())
			Expect(r).ToNot(BeNil())
			r.Close()
			Expect(path).To(Equal(f3))

			sf = reader.fromExternalFile(ctx, "cover.*")
			r, path, err = sf()
			Expect(err).ToNot(HaveOccurred())
			Expect(r).ToNot(BeNil())
			r.Close()
			Expect(path).To(Equal(f1))
		})

		It("respects DiscArtPriority order when both numbered and unnumbered patterns match", func() {
			f1 := createFile("album/cover.png")
			f2 := createFile("album/disc1.jpg")
			reader := &discArtworkReader{
				discNumber:  1,
				imgFiles:    []string{f1, f2},
				discFolders: map[string]bool{filepath.Join(tmpDir, "album"): true},
			}

			ff := reader.fromDiscArtPriority(ctx, nil, "disc*.*, cover.*")
			Expect(ff).To(HaveLen(2))
			r, path, err := ff[0]()
			Expect(err).ToNot(HaveOccurred())
			Expect(path).To(Equal(f2))
			r.Close()

			ff = reader.fromDiscArtPriority(ctx, nil, "cover.*, disc*.*")
			Expect(ff).To(HaveLen(2))
			r, path, err = ff[0]()
			Expect(err).ToNot(HaveOccurred())
			Expect(path).To(Equal(f1))
			r.Close()
		})

		DescribeTable("numbered match wins over shared fallback within a pattern",
			func(discNumber, expectedIdx int) {
				files := []string{
					createFile("album/disc.jpg"),
					createFile("album/disc1.jpg"),
					createFile("album/disc2.jpg"),
				}
				reader := &discArtworkReader{
					discNumber:  discNumber,
					imgFiles:    files,
					discFolders: map[string]bool{filepath.Join(tmpDir, "album"): true},
				}

				sf := reader.fromExternalFile(ctx, "disc*.*")
				r, path, err := sf()
				Expect(err).ToNot(HaveOccurred())
				Expect(r).ToNot(BeNil())
				r.Close()
				Expect(path).To(Equal(files[expectedIdx]))
			},
			Entry("disc 2 picks disc2.jpg over the shared disc.jpg", 2, 2),
			Entry("disc 3 falls back to disc.jpg when no numbered match exists", 3, 0),
		)

		It("tries the next fallback candidate when the first one cannot be opened", func() {
			f1 := createFile("album/cover.jpg")
			f2 := createFile("album/cover.png")
			// Remove f1 so os.Open will fail on it; f2 should still win.
			Expect(os.Remove(f1)).To(Succeed())
			reader := &discArtworkReader{
				discNumber:  1,
				imgFiles:    []string{f1, f2},
				discFolders: map[string]bool{filepath.Join(tmpDir, "album"): true},
			}

			sf := reader.fromExternalFile(ctx, "cover.*")
			r, path, err := sf()
			Expect(err).ToNot(HaveOccurred())
			Expect(r).ToNot(BeNil())
			r.Close()
			Expect(path).To(Equal(f2))
		})

		It("keeps scanning literal-pattern matches so fallback retry still works", func() {
			// Guards against an 'early break on first literal match' optimization.
			// Multiple imgFiles entries can share a basename (symlinks, case-variant
			// duplicates on case-sensitive filesystems). If the loop breaks after
			// recording just the first, the fallback retry cannot recover when
			// that first file is unreadable.
			f1 := createFile("album/stale/cover.png")
			f2 := createFile("album/cover.png")
			Expect(os.Remove(f1)).To(Succeed())
			reader := &discArtworkReader{
				discNumber: 1,
				imgFiles:   []string{f1, f2},
				discFolders: map[string]bool{
					filepath.Join(tmpDir, "album"):       true,
					filepath.Join(tmpDir, "album/stale"): true,
				},
				isMultiFolder: true,
			}

			sf := reader.fromExternalFile(ctx, "cover.png")
			r, path, err := sf()
			Expect(err).ToNot(HaveOccurred())
			Expect(r).ToNot(BeNil())
			r.Close()
			Expect(path).To(Equal(f2))
		})

		DescribeTable("filters by disc number for non-'*' wildcard patterns",
			func(pattern string, discNumber, expectedIdx int) {
				files := []string{
					createFile("album/disc1.jpg"),
					createFile("album/disc2.jpg"),
				}
				reader := &discArtworkReader{
					discNumber:  discNumber,
					imgFiles:    files,
					discFolders: map[string]bool{filepath.Join(tmpDir, "album"): true},
				}

				sf := reader.fromExternalFile(ctx, pattern)
				r, path, err := sf()
				Expect(err).ToNot(HaveOccurred())
				Expect(r).ToNot(BeNil())
				r.Close()
				Expect(path).To(Equal(files[expectedIdx]))
			},
			Entry("disc?.jpg, target disc 1 → disc1.jpg", "disc?.jpg", 1, 0),
			Entry("disc?.jpg, target disc 2 → disc2.jpg", "disc?.jpg", 2, 1),
			Entry("disc[0-9].jpg, target disc 1 → disc1.jpg", "disc[0-9].jpg", 1, 0),
			Entry("disc[0-9].jpg, target disc 2 → disc2.jpg", "disc[0-9].jpg", 2, 1),
		)

		It("matches file without number in multi-folder album by folder", func() {
			f1 := createFile("album/cd1/disc.jpg")
			f2 := createFile("album/cd2/disc.jpg")
			reader := &discArtworkReader{
				discNumber:    1,
				imgFiles:      []string{f1, f2},
				discFolders:   map[string]bool{filepath.Join(tmpDir, "album", "cd1"): true},
				isMultiFolder: true,
			}

			sf := reader.fromExternalFile(ctx, "disc*.*")
			r, path, err := sf()
			Expect(err).ToNot(HaveOccurred())
			Expect(r).ToNot(BeNil())
			r.Close()
			Expect(path).To(Equal(f1))
		})

		It("prefers disc number over folder when number is present", func() {
			// disc2.jpg in cd1 folder should match disc 2, not disc 1
			f1 := createFile("album/cd1/disc2.jpg")
			reader := &discArtworkReader{
				discNumber:    2,
				imgFiles:      []string{f1},
				discFolders:   map[string]bool{filepath.Join(tmpDir, "album", "cd1"): true},
				isMultiFolder: true,
			}

			sf := reader.fromExternalFile(ctx, "disc*.*")
			r, path, err := sf()
			Expect(err).ToNot(HaveOccurred())
			Expect(r).ToNot(BeNil())
			r.Close()
			Expect(path).To(Equal(f1))
		})

		It("does not match disc2.jpg when looking for disc 1", func() {
			f1 := createFile("album/disc2.jpg")
			reader := &discArtworkReader{
				discNumber:  1,
				imgFiles:    []string{f1},
				discFolders: map[string]bool{filepath.Join(tmpDir, "album"): true},
			}

			sf := reader.fromExternalFile(ctx, "disc*.*")
			r, _, _ := sf()
			Expect(r).To(BeNil())
		})
	})

	Describe("fromDiscSubtitle", func() {
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

		It("matches image file whose stem equals the disc subtitle (case-insensitive)", func() {
			f1 := createFile("album/The Blue Disc.jpg")
			reader := &discArtworkReader{
				discNumber: 1,
				imgFiles:   []string{f1},
			}

			sf := reader.fromDiscSubtitle(ctx, "The Blue Disc")
			r, path, err := sf()
			Expect(err).ToNot(HaveOccurred())
			Expect(r).ToNot(BeNil())
			r.Close()
			Expect(path).To(Equal(f1))
		})

		It("matches case-insensitively", func() {
			f1 := createFile("album/bonus tracks.png")
			reader := &discArtworkReader{
				discNumber: 2,
				imgFiles:   []string{f1},
			}

			sf := reader.fromDiscSubtitle(ctx, "Bonus Tracks")
			r, path, err := sf()
			Expect(err).ToNot(HaveOccurred())
			Expect(r).ToNot(BeNil())
			r.Close()
			Expect(path).To(Equal(f1))
		})

		It("returns error when no matching file found", func() {
			f1 := createFile("album/cover.jpg")
			reader := &discArtworkReader{
				discNumber: 1,
				imgFiles:   []string{f1},
			}

			sf := reader.fromDiscSubtitle(ctx, "The Blue Disc")
			_, _, err := sf()
			Expect(err).To(HaveOccurred())
		})

		It("matches first file when multiple extensions exist", func() {
			f1 := createFile("album/The Blue Disc.jpg")
			f2 := createFile("album/The Blue Disc.png")
			reader := &discArtworkReader{
				discNumber: 1,
				imgFiles:   []string{f1, f2},
			}

			sf := reader.fromDiscSubtitle(ctx, "The Blue Disc")
			r, path, err := sf()
			Expect(err).ToNot(HaveOccurred())
			Expect(r).ToNot(BeNil())
			r.Close()
			Expect(path).To(Equal(f1))
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

			It("returns source func for discsubtitle pattern", func() {
				reader.album = model.Album{Discs: model.Discs{2: "Bonus Tracks"}}
				ff := reader.fromDiscArtPriority(context.Background(), nil, "discsubtitle")
				Expect(ff).To(HaveLen(1))
			})

			It("returns no source func for discsubtitle when disc has no subtitle", func() {
				reader.album = model.Album{Discs: model.Discs{2: ""}}
				ff := reader.fromDiscArtPriority(context.Background(), nil, "discsubtitle")
				Expect(ff).To(HaveLen(0))
			})
		})
	})
})
