package plugins

import (
	"os"
	"path/filepath"
	"runtime"

	"github.com/navidrome/navidrome/model"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	experimentalsys "github.com/tetratelabs/wazero/experimental/sys"
	"github.com/tetratelabs/wazero/experimental/sysfs"
)

var _ = Describe("buildMounts", func() {
	var libraries model.Libraries

	BeforeEach(func() {
		libraries = model.Libraries{
			{ID: 1, Path: "/music/library1"},
			{ID: 2, Path: "/music/library2"},
			{ID: 3, Path: "/music/library3"},
		}
	})

	It("mounts all libraries read-only by default", func() {
		Expect(buildMounts(nil, libraries, nil, true, false)).To(Equal([]mount{
			{hostPath: "/music/library1", guestPath: "/libraries/1", readOnly: true},
			{hostPath: "/music/library2", guestPath: "/libraries/2", readOnly: true},
			{hostPath: "/music/library3", guestPath: "/libraries/3", readOnly: true},
		}))
	})

	It("mounts only the selected libraries", func() {
		Expect(buildMounts(nil, libraries, []int{1, 3}, false, false)).To(Equal([]mount{
			{hostPath: "/music/library1", guestPath: "/libraries/1", readOnly: true},
			{hostPath: "/music/library3", guestPath: "/libraries/3", readOnly: true},
		}))
	})

	It("mounts writable when write access is granted", func() {
		Expect(buildMounts(nil, libraries, []int{2}, false, true)).To(Equal([]mount{
			{hostPath: "/music/library2", guestPath: "/libraries/2", readOnly: false},
		}))
	})

	DescribeTable("mounts nothing",
		func(libs model.Libraries, allowedIDs []int, allLibraries bool) {
			Expect(buildMounts(nil, libs, allowedIDs, allLibraries, false)).To(BeEmpty())
		},
		Entry("when no library matches", libraries, []int{99}, false),
		Entry("when there are no libraries", model.Libraries(nil), []int{1}, false),
		Entry("when no library is selected", libraries, nil, false),
	)
})

var _ = Describe("jailedFS", func() {
	var (
		fs         jailedFS
		root       string
		outsideDir string
	)

	BeforeEach(func() {
		tmpDir := GinkgoT().TempDir()
		root = filepath.Join(tmpDir, "root")
		outsideDir = filepath.Join(tmpDir, "outside")
		Expect(os.MkdirAll(root, 0755)).To(Succeed())
		Expect(os.MkdirAll(outsideDir, 0755)).To(Succeed())
		Expect(os.WriteFile(filepath.Join(outsideDir, "secret.txt"), []byte("secret"), 0600)).To(Succeed())

		fs = jailedFS{sysfs.DirFS(root)}
	})

	It("denies creating symlinks", func() {
		Expect(fs.Symlink("../outside", "link")).To(Equal(experimentalsys.EPERM))
	})

	It("denies reading through a path that escapes the root", func() {
		_, errno := fs.OpenFile("../outside/secret.txt", experimentalsys.O_RDONLY, 0)
		Expect(errno).To(Equal(experimentalsys.EPERM))
	})

	It("denies traversal with the host separator", func() {
		_, errno := fs.OpenFile(`..\outside\secret.txt`, experimentalsys.O_RDONLY, 0)

		if runtime.GOOS == "windows" {
			Expect(errno).To(Equal(experimentalsys.EPERM))
		} else {
			// A backslash is a legal filename character on POSIX, not a separator
			Expect(errno).To(Equal(experimentalsys.ENOENT))
		}
	})

	It("denies absolute paths", func() {
		_, errno := fs.OpenFile("/etc/passwd", experimentalsys.O_RDONLY, 0)
		Expect(errno).To(Equal(experimentalsys.EPERM))
	})

	It("denies creating a directory outside the root", func() {
		Expect(fs.Mkdir("../outside/new-dir", 0755)).To(Equal(experimentalsys.EPERM))
		Expect(filepath.Join(outsideDir, "new-dir")).ToNot(BeADirectory())
	})

	It("denies escaping through either side of a rename", func() {
		Expect(fs.Rename("../outside/secret.txt", "stolen.txt")).To(Equal(experimentalsys.EPERM))
		Expect(fs.Rename("inside.txt", "../outside/leaked.txt")).To(Equal(experimentalsys.EPERM))
	})

	// Music libraries commonly symlink in folders from elsewhere; only creating
	// new symlinks is denied, not following the ones already there.
	It("follows symlinks that already exist in the mount", func() {
		if err := os.Symlink(outsideDir, filepath.Join(root, "linked")); err != nil {
			Skip("cannot create symlinks here: " + err.Error()) // Windows without privileges
		}

		_, errno := fs.OpenFile("linked/secret.txt", experimentalsys.O_RDONLY, 0)
		Expect(errno).To(BeZero())
	})

	It("allows access within the root", func() {
		Expect(fs.Mkdir("sub", 0755)).To(BeZero())
		f, errno := fs.OpenFile("sub/file.txt", experimentalsys.O_CREAT|experimentalsys.O_WRONLY, 0600)
		Expect(errno).To(BeZero())
		Expect(f.Close()).To(BeZero())
		Expect(filepath.Join(root, "sub", "file.txt")).To(BeAnExistingFile())
	})
})
