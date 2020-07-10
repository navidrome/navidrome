package core

import (
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/deluan/navidrome/conf"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("File Caches", func() {
	BeforeEach(func() {
		conf.Server.DataFolder, _ = ioutil.TempDir("", "file_caches")
	})
	AfterEach(func() {
		os.RemoveAll(conf.Server.DataFolder)
	})

	Describe("newFileCache", func() {
		It("creates the cache folder", func() {
			Expect(newFileCache("test", "1k", "test", 10)).ToNot(BeNil())

			_, err := os.Stat(filepath.Join(conf.Server.DataFolder, "test"))
			Expect(os.IsNotExist(err)).To(BeFalse())
		})

		It("creates the cache folder with invalid size", func() {
			Expect(newFileCache("test", "abc", "test", 10)).ToNot(BeNil())
		})

		It("returns empty if cache size is '0'", func() {
			Expect(newFileCache("test", "0", "test", 10)).To(BeNil())
		})
	})
})
