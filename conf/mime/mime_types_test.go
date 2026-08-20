package mime

import (
	"mime"
	"testing"

	"github.com/navidrome/navidrome/conf"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestMimeTypes(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Mime Types Suite")
}

var _ = Describe("initMimeTypes", func() {
	BeforeEach(func() {
		conf.Server.DataFolder = conf.NewDir(GinkgoT().TempDir())
		initMimeTypes()
	})

	DescribeTable("registers the type matching the container the transcoder emits",
		func(ext, expected string) {
			Expect(mime.TypeByExtension(ext)).To(HavePrefix(expected))
		},
		Entry("aac is raw ADTS, not MP4", ".aac", "audio/aac"),
		Entry("m4a is AAC inside MP4", ".m4a", "audio/mp4"),
		Entry("mp3", ".mp3", "audio/mpeg"),
		Entry("flac", ".flac", "audio/flac"),
	)
})
