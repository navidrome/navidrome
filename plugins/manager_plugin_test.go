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

var _ = Describe("userAccess", func() {
	DescribeTable("allows",
		func(ids []string, all bool, userID string, want bool) {
			Expect(newUserAccess(ids, all).allows(userID)).To(Equal(want))
		},
		Entry("any user when allUsers is set", nil, true, "any-user", true),
		Entry("nobody when the list is nil", nil, false, "user-1", false),
		Entry("nobody when the list is empty", []string{}, false, "user-1", false),
		Entry("a user in the list", []string{"user-1", "user-2"}, false, "user-1", true),
		Entry("not a user outside the list", []string{"user-1", "user-2"}, false, "user-3", false),
	)

	DescribeTable("configured",
		func(ids []string, all bool, want bool) {
			Expect(newUserAccess(ids, all).configured()).To(Equal(want))
		},
		Entry("when allUsers is set", nil, true, true),
		Entry("when the list has a user", []string{"user-1"}, false, true),
		Entry("not when the list is nil", nil, false, false),
		Entry("not when the list is empty", []string{}, false, false),
	)
})

var _ = Describe("libraryAccess", func() {
	DescribeTable("contains",
		func(ids []int, all bool, libID int, want bool) {
			Expect(newLibraryAccess(ids, all).contains(libID)).To(Equal(want))
		},
		Entry("any library when allLibraries is set", nil, true, 42, true),
		Entry("nothing when the list is nil", nil, false, 1, false),
		Entry("nothing when the list is empty", []int{}, false, 1, false),
		Entry("a library in the list", []int{1, 3}, false, 3, true),
		Entry("not a library outside the list", []int{1, 3}, false, 2, false),
	)

	DescribeTable("configured",
		func(ids []int, all bool, want bool) {
			Expect(newLibraryAccess(ids, all).configured()).To(Equal(want))
		},
		Entry("when allLibraries is set", nil, true, true),
		Entry("when the list has a library", []int{1}, false, true),
		Entry("not when the list is nil", nil, false, false),
		Entry("not when the list is empty", []int{}, false, false),
	)
})
