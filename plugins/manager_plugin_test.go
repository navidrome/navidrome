package plugins

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("plugin", func() {
	Describe("hasLibraryFilesystemAccess", func() {
		manifestWithFS := func(fs bool) *Manifest {
			return &Manifest{
				Permissions: &Permissions{
					Library: &LibraryPermission{Filesystem: fs},
				},
			}
		}

		It("returns false when filesystem permission is not granted", func() {
			p := &plugin{manifest: manifestWithFS(false), allLibraries: true}
			Expect(p.hasLibraryFilesystemAccess(1)).To(BeFalse())
		})

		It("returns false when the manifest has no library permission at all", func() {
			p := &plugin{manifest: &Manifest{}, allLibraries: true}
			Expect(p.hasLibraryFilesystemAccess(1)).To(BeFalse())
		})

		It("returns true for any library when allLibraries is set", func() {
			p := &plugin{manifest: manifestWithFS(true), allLibraries: true}
			Expect(p.hasLibraryFilesystemAccess(1)).To(BeTrue())
			Expect(p.hasLibraryFilesystemAccess(42)).To(BeTrue())
		})

		It("returns true only for libraries in the allowed list", func() {
			p := &plugin{
				manifest:          manifestWithFS(true),
				allowedLibraryIDs: []int{1, 3},
			}
			Expect(p.hasLibraryFilesystemAccess(1)).To(BeTrue())
			Expect(p.hasLibraryFilesystemAccess(3)).To(BeTrue())
			Expect(p.hasLibraryFilesystemAccess(2)).To(BeFalse())
		})

		It("returns false when the allowed list is empty and allLibraries is false", func() {
			p := &plugin{manifest: manifestWithFS(true)}
			Expect(p.hasLibraryFilesystemAccess(1)).To(BeFalse())
		})
	})
})
