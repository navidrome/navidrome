package engine

import (
	"context"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"strings"

	"github.com/deluan/navidrome/conf"
	"github.com/deluan/navidrome/log"
	"github.com/deluan/navidrome/model"
	"github.com/deluan/navidrome/persistence"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("MediaStreamer", func() {

	var streamer MediaStreamer
	var ds model.DataStore
	var tempDir string
	ctx := log.NewContext(nil)

	BeforeSuite(func() {
		conf.Server.EnableDownsampling = true
		tempDir, err := ioutil.TempDir("", "stream_tests")
		if err != nil {
			panic(err)
		}
		conf.Server.DataFolder = tempDir
	})

	BeforeEach(func() {
		ds = &persistence.MockDataStore{}
		ds.MediaFile(ctx).(*persistence.MockMediaFile).SetData(`[{"id": "123", "path": "tests/fixtures/test.mp3", "bitRate": 128}]`, 1)
		streamer = NewMediaStreamer(ds, &fakeFFmpeg{})
	})

	AfterSuite(func() {
		os.RemoveAll(tempDir)
	})

	getFile := func(id string, maxBitRate int, format string) (http.File, error) {
		fs, _ := streamer.NewFileSystem(ctx, maxBitRate, format)
		return fs.Open(id)
	}

	Context("NewFileSystem", func() {
		It("returns a File if format is 'raw'", func() {
			Expect(getFile("123", 0, "raw")).To(BeAssignableToTypeOf(&os.File{}))
		})
		It("returns a File if maxBitRate is 0", func() {
			Expect(getFile("123", 0, "mp3")).To(BeAssignableToTypeOf(&os.File{}))
		})
		It("returns a File if maxBitRate is higher than file bitRate", func() {
			Expect(getFile("123", 256, "mp3")).To(BeAssignableToTypeOf(&os.File{}))
		})
		It("returns a transcodingFile if maxBitRate is lower than file bitRate", func() {
			s, err := getFile("123", 64, "mp3")
			Expect(err).To(BeNil())
			Expect(s).To(BeAssignableToTypeOf(&transcodingFile{}))
			Expect(s.(*transcodingFile).bitRate).To(Equal(64))
		})
		It("returns a File if the transcoding is cached", func() {
			Expect(getFile("123", 64, "mp3")).To(BeAssignableToTypeOf(&os.File{}))
		})
	})
})

type fakeFFmpeg struct {
}

func (ff *fakeFFmpeg) StartTranscoding(ctx context.Context, path string, maxBitRate int, format string) (f io.ReadCloser, err error) {
	return ioutil.NopCloser(strings.NewReader("fake data")), nil
}
