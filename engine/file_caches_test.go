package engine

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
			_, err := newFileCache("test", "1kb", "test", 10)
			Expect(err).To(BeNil())

			_, err = os.Stat(filepath.Join(conf.Server.DataFolder, "test"))
			Expect(os.IsNotExist(err)).To(BeFalse())
		})
	})
})
