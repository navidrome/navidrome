package model_test

import (
	"path"
	"time"

	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/id"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Folder", func() {
	var (
		lib model.Library
	)

	BeforeEach(func() {
		lib = model.Library{
			ID:   1,
			Path: "/music",
		}
	})

	Describe("FolderID", func() {
		It("should generate a consistent ID for a given library and path", func() {
			folderPath := "/music/rock"
			expectedID := id.NewHash("1:/rock")
			Expect(model.FolderID(lib, folderPath)).To(Equal(expectedID))
		})

		It("should trim the library path prefix from the folder path", func() {
			folderPath := "/music/rock"
			Expect(model.FolderID(lib, folderPath)).To(Equal(id.NewHash("1:/rock")))
		})
	})

	Describe("NewFolder", func() {
		It("should create a new SubFolder with the correct attributes", func() {
			folderPath := "rock/metal"
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
