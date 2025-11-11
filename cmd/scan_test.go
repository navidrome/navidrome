package cmd

import (
	"github.com/navidrome/navidrome/scanner"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("parseTargets", func() {
	Context("Valid targets", func() {
		It("parses multiple targets", func() {
			targets, err := parseTargets("1:Music/Rock,2:Jazz,3:Classical/Beethoven")
			Expect(err).ToNot(HaveOccurred())
			Expect(targets).To(HaveLen(3))
			Expect(targets[0]).To(Equal(scanner.ScanTarget{LibraryID: 1, FolderPath: "Music/Rock"}))
			Expect(targets[1]).To(Equal(scanner.ScanTarget{LibraryID: 2, FolderPath: "Jazz"}))
			Expect(targets[2]).To(Equal(scanner.ScanTarget{LibraryID: 3, FolderPath: "Classical/Beethoven"}))
		})

		It("returns error for empty string", func() {
			_, err := parseTargets("")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("no valid targets"))
		})

		// Other test cases are covered in scanner/controller_test.go
	})
})
