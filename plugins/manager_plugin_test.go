package plugins

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("plugin", func() {
	Describe("hasLibraryFilesystemAccess", func() {
		fsManifest := &Manifest{
			Permissions: &Permissions{
				Library: &LibraryPermission{Filesystem: true},
			},
		}

		It("returns false when the manifest does not grant filesystem permission", func() {
			p := &plugin{manifest: &Manifest{}, libraries: newLibraryAccess(nil, true)}
			Expect(p.hasLibraryFilesystemAccess(1)).To(BeFalse())
		})

		It("returns true for any library when allLibraries is set", func() {
			p := &plugin{manifest: fsManifest, libraries: newLibraryAccess(nil, true)}
			Expect(p.hasLibraryFilesystemAccess(1)).To(BeTrue())
			Expect(p.hasLibraryFilesystemAccess(42)).To(BeTrue())
		})

		It("returns true only for libraries in the allowed list", func() {
			p := &plugin{manifest: fsManifest, libraries: newLibraryAccess([]int{1, 3}, false)}
			Expect(p.hasLibraryFilesystemAccess(1)).To(BeTrue())
			Expect(p.hasLibraryFilesystemAccess(3)).To(BeTrue())
			Expect(p.hasLibraryFilesystemAccess(2)).To(BeFalse())
		})
	})
})
