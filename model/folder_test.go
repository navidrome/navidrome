package model_test

import (
	"path"
	"path/filepath"
	"time"

	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/id"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Folder.CoverArtID", func() {
	It("returns empty ArtworkID when folder has no images", func() {
		f := model.Folder{ID: "folder-1"}
		Expect(f.CoverArtID()).To(Equal(model.ArtworkID{}))
		Expect(f.CoverArtID().String()).To(BeEmpty())
	})

	It("returns a folder ArtworkID when folder has images", func() {
		now := time.Now().Truncate(time.Second)
		f := model.Folder{ID: "folder-1", ImageFiles: []string{"cover.jpg"}, ImagesUpdatedAt: now}
		artID := f.CoverArtID()
		Expect(artID.Kind).To(Equal(model.KindFolderArtwork))
		Expect(artID.ID).To(Equal("folder-1"))
		Expect(artID.LastUpdate.Unix()).To(Equal(now.Unix()))
	})

	It("produces a parseable ArtworkID string", func() {
		now := time.Now()
		f := model.Folder{ID: "folder-1", ImageFiles: []string{"cover.jpg"}, ImagesUpdatedAt: now}
		parsed, err := model.ParseArtworkID(f.CoverArtID().String())
		Expect(err).ToNot(HaveOccurred())
		Expect(parsed.Kind).To(Equal(model.KindFolderArtwork))
		Expect(parsed.ID).To(Equal("folder-1"))
	})
})

var _ = Describe("Folder", func() {
	var (
		lib model.Library
	)

	BeforeEach(func() {
		lib = model.Library{
			ID:   1,
			Path: filepath.FromSlash("/music"),
		}
	})

	Describe("FolderID", func() {
		When("the folder path is the library root", func() {
			It("should return the correct folder ID", func() {
				folderPath := lib.Path
				expectedID := id.NewHash("1:.")
				Expect(model.FolderID(lib, folderPath)).To(Equal(expectedID))
			})
		})

		When("the folder path is '.' (library root)", func() {
			It("should return the correct folder ID", func() {
				folderPath := "."
				expectedID := id.NewHash("1:.")
				Expect(model.FolderID(lib, folderPath)).To(Equal(expectedID))
			})
		})

		When("the folder path is relative", func() {
			It("should return the correct folder ID", func() {
				folderPath := "rock"
				expectedID := id.NewHash("1:rock")
				Expect(model.FolderID(lib, folderPath)).To(Equal(expectedID))
			})
		})

		When("the folder path starts with '.'", func() {
			It("should return the correct folder ID", func() {
				folderPath := "./rock"
				expectedID := id.NewHash("1:rock")
				Expect(model.FolderID(lib, folderPath)).To(Equal(expectedID))
			})
		})

		When("the folder path is absolute", func() {
			It("should return the correct folder ID", func() {
				folderPath := filepath.FromSlash("/music/rock")
				expectedID := id.NewHash("1:rock")
				Expect(model.FolderID(lib, folderPath)).To(Equal(expectedID))
			})
		})

		When("the folder has multiple subdirs", func() {
			It("should return the correct folder ID", func() {
				tests.SkipOnWindows("path separator bug (#TBD-path-sep-model)")
				folderPath := filepath.FromSlash("/music/rock/metal")
				expectedID := id.NewHash("1:rock/metal")
				Expect(model.FolderID(lib, folderPath)).To(Equal(expectedID))
			})
		})
	})

	Describe("NewFolder", func() {
		It("should create a new SubFolder with the correct attributes", func() {
			tests.SkipOnWindows("path separator bug (#TBD-path-sep-model)")
			folderPath := filepath.FromSlash("rock/metal")
			folder := model.NewFolder(lib, folderPath)

			Expect(folder.LibraryID).To(Equal(lib.ID))
			Expect(folder.ID).To(Equal(model.FolderID(lib, folderPath)))
			Expect(folder.Path).To(Equal(path.Clean("rock")))
			Expect(folder.Name).To(Equal("metal"))
			Expect(folder.ParentID).To(Equal(model.FolderID(lib, "rock")))
			Expect(folder.ImageFiles).To(BeEmpty())
			Expect(folder.UpdateAt).To(BeTemporally("~", time.Now(), time.Second))
			Expect(folder.CreatedAt).To(BeTemporally("~", time.Now(), time.Second))
		})

		It("should create a new Folder with the correct attributes", func() {
			folderPath := "rock"
			folder := model.NewFolder(lib, folderPath)

			Expect(folder.LibraryID).To(Equal(lib.ID))
			Expect(folder.ID).To(Equal(model.FolderID(lib, folderPath)))
			Expect(folder.Path).To(Equal(path.Clean(".")))
			Expect(folder.Name).To(Equal("rock"))
			Expect(folder.ParentID).To(Equal(model.FolderID(lib, ".")))
			Expect(folder.ImageFiles).To(BeEmpty())
			Expect(folder.UpdateAt).To(BeTemporally("~", time.Now(), time.Second))
			Expect(folder.CreatedAt).To(BeTemporally("~", time.Now(), time.Second))
		})

		It("should handle the root folder correctly", func() {
			folderPath := "."
			folder := model.NewFolder(lib, folderPath)

			Expect(folder.LibraryID).To(Equal(lib.ID))
			Expect(folder.ID).To(Equal(model.FolderID(lib, folderPath)))
			Expect(folder.Path).To(Equal(""))
			Expect(folder.Name).To(Equal("."))
			Expect(folder.ParentID).To(Equal(""))
			Expect(folder.ImageFiles).To(BeEmpty())
			Expect(folder.UpdateAt).To(BeTemporally("~", time.Now(), time.Second))
			Expect(folder.CreatedAt).To(BeTemporally("~", time.Now(), time.Second))
		})
	})
})
