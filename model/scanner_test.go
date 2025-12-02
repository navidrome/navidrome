package model_test

import (
	"github.com/navidrome/navidrome/model"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("ParseTargets", func() {
	It("parses multiple entries in slice", func() {
		targets, err := model.ParseTargets([]string{"1:Music/Rock", "1:Music/Jazz", "2:Classical"})
		Expect(err).ToNot(HaveOccurred())
		Expect(targets).To(HaveLen(3))
		Expect(targets[0].LibraryID).To(Equal(1))
		Expect(targets[0].FolderPath).To(Equal("Music/Rock"))
		Expect(targets[1].LibraryID).To(Equal(1))
		Expect(targets[1].FolderPath).To(Equal("Music/Jazz"))
		Expect(targets[2].LibraryID).To(Equal(2))
		Expect(targets[2].FolderPath).To(Equal("Classical"))
	})

	It("handles empty folder paths", func() {
		targets, err := model.ParseTargets([]string{"1:", "2:"})
		Expect(err).ToNot(HaveOccurred())
		Expect(targets).To(HaveLen(2))
		Expect(targets[0].FolderPath).To(Equal(""))
		Expect(targets[1].FolderPath).To(Equal(""))
	})

	It("trims whitespace from entries", func() {
		targets, err := model.ParseTargets([]string{"  1:Music/Rock", " 2:Classical "})
		Expect(err).ToNot(HaveOccurred())
		Expect(targets).To(HaveLen(2))
		Expect(targets[0].LibraryID).To(Equal(1))
		Expect(targets[0].FolderPath).To(Equal("Music/Rock"))
		Expect(targets[1].LibraryID).To(Equal(2))
		Expect(targets[1].FolderPath).To(Equal("Classical"))
	})

	It("skips empty strings", func() {
		targets, err := model.ParseTargets([]string{"1:Music/Rock", "", "2:Classical"})
		Expect(err).ToNot(HaveOccurred())
		Expect(targets).To(HaveLen(2))
	})

	It("handles paths with colons", func() {
		targets, err := model.ParseTargets([]string{"1:C:/Music/Rock", "2:/path:with:colons"})
		Expect(err).ToNot(HaveOccurred())
		Expect(targets).To(HaveLen(2))
		Expect(targets[0].FolderPath).To(Equal("C:/Music/Rock"))
		Expect(targets[1].FolderPath).To(Equal("/path:with:colons"))
	})

	It("returns error for invalid format without colon", func() {
		_, err := model.ParseTargets([]string{"1Music/Rock"})
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("invalid target format"))
	})

	It("returns error for non-numeric library ID", func() {
		_, err := model.ParseTargets([]string{"abc:Music/Rock"})
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("invalid library ID"))
	})

	It("returns error for negative library ID", func() {
		_, err := model.ParseTargets([]string{"-1:Music/Rock"})
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("invalid library ID"))
	})

	It("returns error for zero library ID", func() {
		_, err := model.ParseTargets([]string{"0:Music/Rock"})
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("invalid library ID"))
	})

	It("returns error for empty input", func() {
		_, err := model.ParseTargets([]string{})
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("no valid targets found"))
	})

	It("returns error for all empty strings", func() {
		_, err := model.ParseTargets([]string{"", "  ", ""})
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("no valid targets found"))
	})
})
