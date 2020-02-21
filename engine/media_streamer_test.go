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
	"gopkg.in/djherbis/fscache.v0"
)

var _ = Describe("MediaStreamer", func() {

	var streamer MediaStreamer
	var ds model.DataStore
	ctx := log.NewContext(nil)

	BeforeEach(func() {
		conf.Server.EnableDownsampling = true
		fs := fscache.NewMemFs()
		cache, _ := fscache.NewCache(fs, nil)
		ds = &persistence.MockDataStore{}
		ds.MediaFile(ctx).(*persistence.MockMediaFile).SetData(`[{"id": "123", "path": "tests/fixtures/test.mp3", "bitRate": 128}]`, 1)
		streamer = NewMediaStreamer(ds, &fakeFFmpeg{}, cache)
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
	})
})

type fakeFFmpeg struct {
}

func (ff *fakeFFmpeg) StartTranscoding(ctx context.Context, path string, maxBitRate int, format string) (f io.ReadCloser, err error) {
	return ioutil.NopCloser(strings.NewReader("fake data")), nil
}
