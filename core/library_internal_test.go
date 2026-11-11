package core

import (
	"errors"
	"fmt"
	"io/fs"
	"syscall"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("classifyLibraryPathError", func() {
	DescribeTable("maps a filesystem error to a validation message key",
		func(err error, expected string) {
			Expect(classifyLibraryPathError(err)).To(Equal(expected))
		},
		Entry("missing path", fs.ErrNotExist, "resources.library.validation.pathNotFound"),
		Entry("wrapped missing path", fmt.Errorf("stat /music: %w", fs.ErrNotExist), "resources.library.validation.pathNotFound"),
		Entry("permission denied", fs.ErrPermission, "resources.library.validation.pathNotAccessible"),
		Entry("wrapped permission denied", fmt.Errorf("open /music: %w", fs.ErrPermission), "resources.library.validation.pathNotAccessible"),
		Entry("file instead of a directory on Unix",
			&fs.PathError{Op: "stat", Path: "/music/song.mp3/.", Err: syscall.ENOTDIR},
			"resources.library.validation.pathNotDirectory"),
		Entry("file instead of a directory on Windows",
			&fs.PathError{Op: "open", Path: `C:\music\song.mp3`, Err: errors.New("The directory name is invalid.")},
			"resources.library.validation.pathNotDirectory"),
		Entry("any other error", errors.New("input/output error"), "resources.library.validation.pathInvalid"),
	)
})
