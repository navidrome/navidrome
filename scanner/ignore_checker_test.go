package scanner

import (
	"context"
	"testing/fstest"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("IgnoreChecker", func() {
	Describe("loadPatternsFromFolder", func() {
		var ic *IgnoreChecker
		var ctx context.Context

		BeforeEach(func() {
			ctx = context.Background()
		})

		Context("when .ndignore file does not exist", func() {
			It("should return empty patterns", func() {
				fsys := fstest.MapFS{}
				ic = newIgnoreChecker(fsys)
				patterns := ic.loadPatternsFromFolder(ctx, ".")
				Expect(patterns).To(BeEmpty())
			})
		})

		Context("when .ndignore file is empty", func() {
			It("should return wildcard to ignore everything", func() {
				fsys := fstest.MapFS{
					".ndignore": &fstest.MapFile{Data: []byte("")},
				}
				ic = newIgnoreChecker(fsys)
				patterns := ic.loadPatternsFromFolder(ctx, ".")
				Expect(patterns).To(Equal([]string{"**/*"}))
			})
		})

		DescribeTable("parsing .ndignore content",
			func(content string, expectedPatterns []string) {
				fsys := fstest.MapFS{
					".ndignore": &fstest.MapFile{Data: []byte(content)},
				}
				ic = newIgnoreChecker(fsys)
				patterns := ic.loadPatternsFromFolder(ctx, ".")
				Expect(patterns).To(Equal(expectedPatterns))
			},
			Entry("single pattern", "*.txt", []string{"*.txt"}),
			Entry("multiple patterns", "*.txt\n*.log", []string{"*.txt", "*.log"}),
			Entry("with comments", "# comment\n*.txt\n# another\n*.log", []string{"*.txt", "*.log"}),
			Entry("with empty lines", "*.txt\n\n*.log\n\n", []string{"*.txt", "*.log"}),
			Entry("mixed content", "# header\n\n*.txt\n# middle\n*.log\n\n", []string{"*.txt", "*.log"}),
			Entry("only comments and empty lines", "# comment\n\n# another\n", []string{"**/*"}),
			Entry("trailing newline", "*.txt\n*.log\n", []string{"*.txt", "*.log"}),
			Entry("directory pattern", "temp/", []string{"temp/"}),
			Entry("wildcard pattern", "**/*.mp3", []string{"**/*.mp3"}),
			Entry("multiple wildcards", "**/*.mp3\n**/*.flac\n*.log", []string{"**/*.mp3", "**/*.flac", "*.log"}),
			Entry("negation pattern", "!important.txt", []string{"!important.txt"}),
			Entry("comment with hash not at start is pattern", "not#comment", []string{"not#comment"}),
			Entry("whitespace-only lines skipped", "*.txt\n   \n*.log\n\t\n", []string{"*.txt", "*.log"}),
			Entry("patterns with whitespace trimmed", "  *.txt  \n\t*.log\t", []string{"*.txt", "*.log"}),
		)
	})

	Describe("Push and Pop", func() {
		var ic *IgnoreChecker
		var fsys fstest.MapFS
		var ctx context.Context

		BeforeEach(func() {
			ctx = context.Background()
			fsys = fstest.MapFS{
				".ndignore":         &fstest.MapFile{Data: []byte("*.txt")},
				"folder1/.ndignore": &fstest.MapFile{Data: []byte("*.mp3")},
				"folder2/.ndignore": &fstest.MapFile{Data: []byte("*.flac")},
			}
			ic = newIgnoreChecker(fsys)
		})

		Context("Push", func() {
			It("should add patterns to stack", func() {
				err := ic.Push(ctx, ".")
				Expect(err).ToNot(HaveOccurred())
				Expect(len(ic.patternStack)).To(Equal(1))
				Expect(ic.currentPatterns).To(ContainElement("*.txt"))
			})

			It("should compile matcher after push", func() {
				err := ic.Push(ctx, ".")
				Expect(err).ToNot(HaveOccurred())
				Expect(ic.matcher).ToNot(BeNil())
			})

			It("should accumulate patterns from multiple levels", func() {
				err := ic.Push(ctx, ".")
				Expect(err).ToNot(HaveOccurred())
				err = ic.Push(ctx, "folder1")
				Expect(err).ToNot(HaveOccurred())
				Expect(len(ic.patternStack)).To(Equal(2))
				Expect(ic.currentPatterns).To(ConsistOf("*.txt", "*.mp3"))
			})

			It("should handle push when no .ndignore exists", func() {
				err := ic.Push(ctx, "nonexistent")
				Expect(err).ToNot(HaveOccurred())
				Expect(len(ic.patternStack)).To(Equal(1))
				Expect(ic.currentPatterns).To(BeEmpty())
			})
		})

		Context("Pop", func() {
			It("should remove most recent patterns", func() {
				err := ic.Push(ctx, ".")
				Expect(err).ToNot(HaveOccurred())
				err = ic.Push(ctx, "folder1")
				Expect(err).ToNot(HaveOccurred())
				ic.Pop()
				Expect(len(ic.patternStack)).To(Equal(1))
				Expect(ic.currentPatterns).To(Equal([]string{"*.txt"}))
			})

			It("should handle Pop on empty stack gracefully", func() {
				Expect(func() { ic.Pop() }).ToNot(Panic())
				Expect(ic.patternStack).To(BeEmpty())
			})

			It("should set matcher to nil when all patterns popped", func() {
				err := ic.Push(ctx, ".")
				Expect(err).ToNot(HaveOccurred())
				Expect(ic.matcher).ToNot(BeNil())
				ic.Pop()
				Expect(ic.matcher).To(BeNil())
			})

			It("should update matcher after pop", func() {
				err := ic.Push(ctx, ".")
				Expect(err).ToNot(HaveOccurred())
				err = ic.Push(ctx, "folder1")
				Expect(err).ToNot(HaveOccurred())
				matcher1 := ic.matcher
				ic.Pop()
				matcher2 := ic.matcher
				Expect(matcher1).ToNot(Equal(matcher2))
			})
		})

		Context("multiple Push/Pop cycles", func() {
			It("should maintain correct state through cycles", func() {
				err := ic.Push(ctx, ".")
				Expect(err).ToNot(HaveOccurred())
				Expect(ic.currentPatterns).To(Equal([]string{"*.txt"}))

				err = ic.Push(ctx, "folder1")
				Expect(err).ToNot(HaveOccurred())
				Expect(ic.currentPatterns).To(ConsistOf("*.txt", "*.mp3"))

				ic.Pop()
				Expect(ic.currentPatterns).To(Equal([]string{"*.txt"}))

				err = ic.Push(ctx, "folder2")
				Expect(err).ToNot(HaveOccurred())
				Expect(ic.currentPatterns).To(ConsistOf("*.txt", "*.flac"))

				ic.Pop()
				Expect(ic.currentPatterns).To(Equal([]string{"*.txt"}))

				ic.Pop()
				Expect(ic.currentPatterns).To(BeEmpty())
			})
		})
	})

	Describe("PushAllParents", func() {
		var ic *IgnoreChecker
		var ctx context.Context

		BeforeEach(func() {
			ctx = context.Background()
			fsys := fstest.MapFS{
				".ndignore":                         &fstest.MapFile{Data: []byte("root.txt")},
				"folder1/.ndignore":                 &fstest.MapFile{Data: []byte("level1.txt")},
				"folder1/folder2/.ndignore":         &fstest.MapFile{Data: []byte("level2.txt")},
				"folder1/folder2/folder3/.ndignore": &fstest.MapFile{Data: []byte("level3.txt")},
			}
			ic = newIgnoreChecker(fsys)
		})

		DescribeTable("loading parent patterns",
			func(targetPath string, expectedStackDepth int, expectedPatterns []string) {
				err := ic.PushAllParents(ctx, targetPath)
				Expect(err).ToNot(HaveOccurred())
				Expect(len(ic.patternStack)).To(Equal(expectedStackDepth))
				Expect(ic.currentPatterns).To(ConsistOf(expectedPatterns))
			},
			Entry("root path", ".", 1, []string{"root.txt"}),
			Entry("empty path", "", 1, []string{"root.txt"}),
			Entry("single level", "folder1", 2, []string{"root.txt", "level1.txt"}),
			Entry("two levels", "folder1/folder2", 3, []string{"root.txt", "level1.txt", "level2.txt"}),
			Entry("three levels", "folder1/folder2/folder3", 4, []string{"root.txt", "level1.txt", "level2.txt", "level3.txt"}),
		)

		It("should only compile patterns once at the end", func() {
			// This is more of a behavioral test - we verify the matcher is not nil after PushAllParents
			err := ic.PushAllParents(ctx, "folder1/folder2")
			Expect(err).ToNot(HaveOccurred())
			Expect(ic.matcher).ToNot(BeNil())
		})

		It("should handle paths with dot", func() {
			err := ic.PushAllParents(ctx, "./folder1")
			Expect(err).ToNot(HaveOccurred())
			Expect(len(ic.patternStack)).To(Equal(2))
		})

		Context("when some parent folders have no .ndignore", func() {
			BeforeEach(func() {
				fsys := fstest.MapFS{
					".ndignore":                 &fstest.MapFile{Data: []byte("root.txt")},
					"folder1/folder2/.ndignore": &fstest.MapFile{Data: []byte("level2.txt")},
				}
				ic = newIgnoreChecker(fsys)
			})

			It("should still push all parent levels", func() {
				err := ic.PushAllParents(ctx, "folder1/folder2")
				Expect(err).ToNot(HaveOccurred())
				Expect(len(ic.patternStack)).To(Equal(3)) // root, folder1 (empty), folder2
				Expect(ic.currentPatterns).To(ConsistOf("root.txt", "level2.txt"))
			})
		})
	})

	Describe("ShouldIgnore", func() {
		var ic *IgnoreChecker
		var ctx context.Context

		BeforeEach(func() {
			ctx = context.Background()
		})

		Context("with no patterns loaded", func() {
			It("should not ignore any path", func() {
				fsys := fstest.MapFS{}
				ic = newIgnoreChecker(fsys)
				Expect(ic.ShouldIgnore(ctx, "anything.txt")).To(BeFalse())
				Expect(ic.ShouldIgnore(ctx, "folder/file.mp3")).To(BeFalse())
			})
		})

		Context("special paths", func() {
			BeforeEach(func() {
				fsys := fstest.MapFS{
					".ndignore": &fstest.MapFile{Data: []byte("**/*")},
				}
				ic = newIgnoreChecker(fsys)
				err := ic.Push(ctx, ".")
				Expect(err).ToNot(HaveOccurred())
			})

			It("should never ignore root or empty paths", func() {
				Expect(ic.ShouldIgnore(ctx, "")).To(BeFalse())
				Expect(ic.ShouldIgnore(ctx, ".")).To(BeFalse())
			})

			It("should ignore all other paths with wildcard", func() {
				Expect(ic.ShouldIgnore(ctx, "file.txt")).To(BeTrue())
				Expect(ic.ShouldIgnore(ctx, "folder/file.mp3")).To(BeTrue())
			})
		})

		DescribeTable("pattern matching",
			func(pattern string, path string, shouldMatch bool) {
				fsys := fstest.MapFS{
					".ndignore": &fstest.MapFile{Data: []byte(pattern)},
				}
				ic = newIgnoreChecker(fsys)
				err := ic.Push(ctx, ".")
				Expect(err).ToNot(HaveOccurred())
				Expect(ic.ShouldIgnore(ctx, path)).To(Equal(shouldMatch))
			},
			Entry("glob match", "*.txt", "file.txt", true),
			Entry("glob no match", "*.txt", "file.mp3", false),
			Entry("directory pattern match", "tmp/", "tmp/file.txt", true),
			Entry("directory pattern no match", "tmp/", "temporary/file.txt", false),
			Entry("nested glob match", "**/*.log", "deep/nested/file.log", true),
			Entry("nested glob no match", "**/*.log", "deep/nested/file.txt", false),
			Entry("specific file match", "ignore.me", "ignore.me", true),
			Entry("specific file no match", "ignore.me", "keep.me", false),
			Entry("wildcard all", "**/*", "any/path/file.txt", true),
			Entry("nested specific match", "temp/*", "temp/cache.db", true),
			Entry("nested specific no match", "temp/*", "temporary/cache.db", false),
		)

		Context("with multiple patterns", func() {
			BeforeEach(func() {
				fsys := fstest.MapFS{
					".ndignore": &fstest.MapFile{Data: []byte("*.txt\n*.log\ntemp/")},
				}
				ic = newIgnoreChecker(fsys)
				err := ic.Push(ctx, ".")
				Expect(err).ToNot(HaveOccurred())
			})

			It("should match any of the patterns", func() {
				Expect(ic.ShouldIgnore(ctx, "file.txt")).To(BeTrue())
				Expect(ic.ShouldIgnore(ctx, "debug.log")).To(BeTrue())
				Expect(ic.ShouldIgnore(ctx, "temp/cache")).To(BeTrue())
				Expect(ic.ShouldIgnore(ctx, "music.mp3")).To(BeFalse())
			})
		})
	})
})
