package core_test

import (
	"path/filepath"

	"github.com/navidrome/navidrome/core"
	"github.com/navidrome/navidrome/model"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Inspect", func() {
	var fixtures string

	BeforeEach(func() {
		var err error
		fixtures, err = filepath.Abs(filepath.Join("tests", "fixtures"))
		Expect(err).ToNot(HaveOccurred())
	})

	It("maps the file with the library-relative path the scanner uses", func() {
		lib := model.Library{ID: 2, Path: filepath.Dir(fixtures), PIDAlbum: "folder"}
		out, err := core.Inspect(filepath.Join(fixtures, "test.mp3"), lib, "")
		Expect(err).ToNot(HaveOccurred())
		Expect(out.MappedTags.Path).To(Equal("fixtures/test.mp3"))
		Expect(out.MappedTags.LibraryID).To(Equal(2))
	})

	It("gives the same IDs for relative and absolute paths", func() {
		lib := model.Library{ID: 2, Path: filepath.Dir(fixtures), PIDAlbum: "folder"}
		abs, err := core.Inspect(filepath.Join(fixtures, "test.mp3"), lib, "")
		Expect(err).ToNot(HaveOccurred())
		rel, err := core.Inspect(filepath.Join("tests", "fixtures", "test.mp3"), lib, "")
		Expect(err).ToNot(HaveOccurred())
		Expect(rel.MappedTags.AlbumID).To(Equal(abs.MappedTags.AlbumID))
		Expect(rel.MappedTags.PID).To(Equal(abs.MappedTags.PID))
	})

	It("keeps the given path for a file outside the library", func() {
		filePath := filepath.Join(fixtures, "test.mp3")
		out, err := core.Inspect(filePath, model.Library{ID: model.DefaultLibraryID}, "")
		Expect(err).ToNot(HaveOccurred())
		Expect(out.MappedTags.Path).To(Equal(filePath))
	})
})
