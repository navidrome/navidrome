package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("writePlaylist", func() {
	const m3u = "#EXTM3U\n#PLAYLIST:DJ Wave\n#EXTINF:364,Bel Canto - Dreaming Girl\n"
	plsFile := filepath.Join(os.TempDir(), fmt.Sprintf("navidrome-pls-%d.m3u8", os.Getpid()))

	BeforeEach(func() {
		DeferCleanup(func() { _ = os.Remove(plsFile) })
	})

	DescribeTable("writes the playlist to exactly one destination",
		func(file, wantStream, wantFile string) {
			var out strings.Builder

			writePlaylist(m3u, &out, file)

			written, _ := os.ReadFile(plsFile)
			Expect(out.String()).To(Equal(wantStream))
			Expect(string(written)).To(Equal(wantFile))
		},
		Entry("no file name writes to the stream", "", m3u, ""),
		Entry("a dash writes to the stream", "-", m3u, ""),
		Entry("a path writes to the file", plsFile, "", m3u),
	)
})
