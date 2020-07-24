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

	Describe("NewFileCache", func() {
		It("creates the cache folder", func() {
			Expect(NewFileCache("test", "1k", "test", 10, nil)).ToNot(BeNil())

			_, err := os.Stat(filepath.Join(conf.Server.DataFolder, "test"))
			Expect(os.IsNotExist(err)).To(BeFalse())
		})

		It("creates the cache folder with invalid size", func() {
			fc, err := NewFileCache("test", "abc", "test", 10, nil)
			Expect(err).To(BeNil())
			Expect(fc.cache).ToNot(BeNil())
			Expect(fc.disabled).To(BeFalse())
		})

		It("returns empty if cache size is '0'", func() {
			fc, err := NewFileCache("test", "0", "test", 10, nil)
			Expect(err).To(BeNil())
			Expect(fc.cache).To(BeNil())
			Expect(fc.disabled).To(BeTrue())
		})
	})

	Describe("FileCache", func() {

	})
})
