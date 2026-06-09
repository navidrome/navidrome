package scanner

import (
	"context"
	"os"
	"strings"

	"github.com/navidrome/navidrome/model"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("targetArguments", func() {
	var ctx context.Context

	BeforeEach(func() {
		ctx = GinkgoT().Context()
	})

	Context("with small target list", func() {
		It("returns command-line arguments for single target", func() {
			targets := []model.ScanTarget{
				{LibraryID: 1, FolderPath: "Music/Rock"},
			}

			args, cleanup, err := targetArguments(ctx, targets, argLengthThreshold)
			Expect(err).ToNot(HaveOccurred())
			defer cleanup()
			Expect(args).To(Equal([]string{"-t", "1:Music/Rock"}))
		})

		It("returns command-line arguments for multiple targets", func() {
			targets := []model.ScanTarget{
				{LibraryID: 1, FolderPath: "Music/Rock"},
				{LibraryID: 2, FolderPath: "Music/Jazz"},
				{LibraryID: 3, FolderPath: "Classical"},
			}

			args, cleanup, err := targetArguments(ctx, targets, argLengthThreshold)
			Expect(err).ToNot(HaveOccurred())
			defer cleanup()
			Expect(args).To(Equal([]string{
				"-t", "1:Music/Rock",
				"-t", "2:Music/Jazz",
				"-t", "3:Classical",
			}))
		})

		It("handles targets with special characters", func() {
			targets := []model.ScanTarget{
				{LibraryID: 1, FolderPath: "Music/Rock & Roll"},
				{LibraryID: 2, FolderPath: "Music/Jazz (Modern)"},
			}

			args, cleanup, err := targetArguments(ctx, targets, argLengthThreshold)
			Expect(err).ToNot(HaveOccurred())
			defer cleanup()
			Expect(args).To(Equal([]string{
				"-t", "1:Music/Rock & Roll",
				"-t", "2:Music/Jazz (Modern)",
			}))
		})
	})

	Context("with large target list exceeding threshold", func() {
		It("returns --target-file argument when exceeding threshold", func() {
			// Create enough targets to exceed the threshold
			var targets []model.ScanTarget
			for i := 1; i <= 600; i++ {
				targets = append(targets, model.ScanTarget{
					LibraryID:  1,
					FolderPath: "Music/VeryLongFolderPathToSimulateRealScenario/SubFolder",
				})
			}

			args, cleanup, err := targetArguments(ctx, targets, argLengthThreshold)
			Expect(err).ToNot(HaveOccurred())
			defer cleanup()
			Expect(args).To(HaveLen(2))
			Expect(args[0]).To(Equal("--target-file"))

			// Verify the file exists and has correct format
			filePath := args[1]
			Expect(filePath).To(ContainSubstring("navidrome-scan-targets-"))
			Expect(filePath).To(HaveSuffix(".txt"))

			// Verify file actually exists
			_, err = os.Stat(filePath)
			Expect(err).ToNot(HaveOccurred())
		})

		It("creates temp file with correct format", func() {
			// Use custom threshold to easily exceed it
			targets := []model.ScanTarget{
				{LibraryID: 1, FolderPath: "Music/Rock"},
				{LibraryID: 2, FolderPath: "Music/Jazz"},
				{LibraryID: 3, FolderPath: "Classical"},
			}

			// Set threshold very low to force file usage
			args, cleanup, err := targetArguments(ctx, targets, 10)
			Expect(err).ToNot(HaveOccurred())
			defer cleanup()
			Expect(args[0]).To(Equal("--target-file"))

			// Verify file exists with correct format
			filePath := args[1]
			Expect(filePath).To(ContainSubstring("navidrome-scan-targets-"))
			Expect(filePath).To(HaveSuffix(".txt"))

			// Verify file content
			content, err := os.ReadFile(filePath)
			Expect(err).ToNot(HaveOccurred())
			lines := strings.Split(strings.TrimSpace(string(content)), "\n")
			Expect(lines).To(HaveLen(3))
			Expect(lines[0]).To(Equal("1:Music/Rock"))
			Expect(lines[1]).To(Equal("2:Music/Jazz"))
			Expect(lines[2]).To(Equal("3:Classical"))
		})
	})

	Context("edge cases", func() {
		It("handles empty target list", func() {
			var targets []model.ScanTarget

			args, cleanup, err := targetArguments(ctx, targets, argLengthThreshold)
			Expect(err).ToNot(HaveOccurred())
			defer cleanup()
			Expect(args).To(BeEmpty())
		})

		It("uses command-line args when exactly at threshold", func() {
			// Create targets that are exactly at threshold
			targets := []model.ScanTarget{
				{LibraryID: 1, FolderPath: "Music"},
			}

			// Estimate length should be 11 bytes
			estimatedLength := estimateArgLength(targets)

			args, cleanup, err := targetArguments(ctx, targets, estimatedLength)
			Expect(err).ToNot(HaveOccurred())
			defer cleanup()
			Expect(args).To(Equal([]string{"-t", "1:Music"}))
		})

		It("uses file when one byte over threshold", func() {
			targets := []model.ScanTarget{
				{LibraryID: 1, FolderPath: "Music"},
			}

			// Set threshold just below the estimated length
			estimatedLength := estimateArgLength(targets)
			args, cleanup, err := targetArguments(ctx, targets, estimatedLength-1)
			Expect(err).ToNot(HaveOccurred())
			defer cleanup()
			Expect(args[0]).To(Equal("--target-file"))
		})
	})
})
