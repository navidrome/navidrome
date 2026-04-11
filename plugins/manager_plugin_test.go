package plugins

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("plugin", func() {
	Describe("hasLibraryFilesystemAccess", func() {
		It("returns false when filesystem permission is not granted", func() {
			p := &plugin{hasFilesystemPerm: false, allLibraries: true}
			Expect(p.hasLibraryFilesystemAccess(1)).To(BeFalse())
		})

		It("returns true for any library when allLibraries is set", func() {
			p := &plugin{hasFilesystemPerm: true, allLibraries: true}
			Expect(p.hasLibraryFilesystemAccess(1)).To(BeTrue())
			Expect(p.hasLibraryFilesystemAccess(42)).To(BeTrue())
		})

		It("returns true only for libraries in the allowed list", func() {
			p := &plugin{
				hasFilesystemPerm: true,
				allowedLibraryIDs: []int{1, 3},
			}
			Expect(p.hasLibraryFilesystemAccess(1)).To(BeTrue())
			Expect(p.hasLibraryFilesystemAccess(3)).To(BeTrue())
			Expect(p.hasLibraryFilesystemAccess(2)).To(BeFalse())
		})

		It("returns false when the allowed list is empty and allLibraries is false", func() {
			p := &plugin{hasFilesystemPerm: true}
			Expect(p.hasLibraryFilesystemAccess(1)).To(BeFalse())
		})
	})
})
