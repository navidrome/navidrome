package cmd

import (
	"github.com/navidrome/navidrome/scanner"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("parseTargets", func() {
	Context("Valid targets", func() {
		It("parses a single target", func() {
			targets, err := parseTargets("1:Music/Rock")
			Expect(err).ToNot(HaveOccurred())
			Expect(targets).To(HaveLen(1))
			Expect(targets[0].LibraryID).To(Equal(1))
			Expect(targets[0].FolderPath).To(Equal("Music/Rock"))
		})

		It("parses multiple targets", func() {
			targets, err := parseTargets("1:Music/Rock,2:Jazz,3:Classical/Beethoven")
			Expect(err).ToNot(HaveOccurred())
			Expect(targets).To(HaveLen(3))
			Expect(targets[0]).To(Equal(scanner.ScanTarget{LibraryID: 1, FolderPath: "Music/Rock"}))
			Expect(targets[1]).To(Equal(scanner.ScanTarget{LibraryID: 2, FolderPath: "Jazz"}))
			Expect(targets[2]).To(Equal(scanner.ScanTarget{LibraryID: 3, FolderPath: "Classical/Beethoven"}))
		})

		It("handles targets with spaces around commas", func() {
			targets, err := parseTargets("1:Music/Rock And Roll, 2:Jazz , 3:Classical")
			Expect(err).ToNot(HaveOccurred())
			Expect(targets).To(HaveLen(3))
			Expect(targets[0].FolderPath).To(Equal("Music/Rock And Roll"))
		})

		It("handles paths with colons after the first colon", func() {
			targets, err := parseTargets("1:C:/Music/Rock")
			Expect(err).ToNot(HaveOccurred())
			Expect(targets).To(HaveLen(1))
			Expect(targets[0].LibraryID).To(Equal(1))
			Expect(targets[0].FolderPath).To(Equal("C:/Music/Rock"))
		})

		It("handles empty folder paths", func() {
			targets, err := parseTargets("1:,2:")
			Expect(err).ToNot(HaveOccurred())
			Expect(targets).To(HaveLen(2))
			Expect(targets[0].FolderPath).To(BeEmpty())
			Expect(targets[1].FolderPath).To(BeEmpty())
		})
	})

	Context("Invalid targets", func() {
		It("returns error for empty string", func() {
			_, err := parseTargets("")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("no valid targets"))
		})

		It("returns error for missing colon", func() {
			_, err := parseTargets("1Music")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("invalid target format"))
		})

		It("returns error for invalid library ID", func() {
			_, err := parseTargets("abc:Music")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("invalid library ID"))
		})

		It("handles negative library ID", func() {
			targets, err := parseTargets("-1:Music")
			Expect(err).ToNot(HaveOccurred()) // Actually valid - strconv.Atoi accepts negative numbers
			Expect(targets[0].LibraryID).To(Equal(-1))
		})

		It("handles only whitespace", func() {
			_, err := parseTargets("   ,  ,  ")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("no valid targets"))
		})
	})
})
