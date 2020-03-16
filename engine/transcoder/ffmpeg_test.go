package transcoder

import (
	"testing"

	"github.com/deluan/navidrome/log"
	"github.com/deluan/navidrome/tests"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestTranscoder(t *testing.T) {
	tests.Init(t, false)
	log.SetLevel(log.LevelCritical)
	RegisterFailHandler(Fail)
	RunSpecs(t, "Transcoder Suite")
}

var _ = Describe("createTranscodeCommand", func() {
	It("creates a valid command line", func() {
		cmd, args := createTranscodeCommand("ffmpeg -i %s -b:a %bk mp3 -", "/music library/file.mp3", 123, "")
		Expect(cmd).To(Equal("ffmpeg"))
		Expect(args).To(Equal([]string{"-i", "/music library/file.mp3", "-b:a", "123k", "mp3", "-"}))
	})
})
