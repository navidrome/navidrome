package local

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"time"

	"github.com/djherbis/times"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/metadata"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.senan.xyz/taglib"
)

const genreComment = " /* Ambient / Soundtrack / Sleep / Dreamy */"

var _ = Describe("Comment genre writeback", func() {
	DescribeTable("extracts only a complete Rekordbox block",
		func(comment string, existing, want []string) {
			Expect(commentGenres([]string{comment}, existing)).To(Equal(want))
		},
		Entry("trims and merges", "URL "+genreComment+" trailing text", []string{"Electronic"}, []string{"Electronic", "Ambient", "Soundtrack", "Sleep", "Dreamy"}),
		Entry("skips empty and duplicate genres", "/* ambient / / Sleep / sleep */", []string{"Ambient"}, []string{"Ambient", "Sleep"}),
		Entry("ignores unclosed blocks", "/* Ambient / Sleep", []string{"Electronic"}, []string{"Electronic"}),
		Entry("ignores free text", "Ambient / Sleep", []string{"Electronic"}, []string{"Electronic"}),
	)

	It("updates the FLAC genres once without changing timestamps, inode, or audio", func() {
		dir := GinkgoT().TempDir()
		file := filepath.Join(dir, "track.flac")
		audio, err := os.ReadFile("tests/fixtures/comment-genres.flac")
		Expect(err).NotTo(HaveOccurred())
		Expect(os.WriteFile(file, audio, 0600)).To(Succeed())
		lfs := &localFS{root: dir}
		mtime := time.Now().Add(-24 * time.Hour).Truncate(time.Millisecond)
		Expect(os.Chtimes(file, mtime, mtime)).To(Succeed())
		before, err := os.Stat(file)
		Expect(err).NotTo(HaveOccurred())
		beforeTimes, err := times.Stat(file)
		Expect(err).NotTo(HaveOccurred())

		info := metadata.Info{
			FileInfo: localFileInfo{FileInfo: before, path: file},
			Tags:     model.RawTags{"comment": {genreComment}, "genre": {"Electronic"}},
		}
		Expect(lfs.WriteCommentGenres("track.flac", &info)).To(Succeed())
		wantGenres := []string{"Electronic", "Ambient", "Soundtrack", "Sleep", "Dreamy"}
		tags, err := taglib.ReadTags(file)
		Expect(err).NotTo(HaveOccurred())
		Expect(tags["GENRE"]).To(Equal(wantGenres))
		Expect(tags["COMMENT"]).To(Equal([]string{genreComment}))
		Expect(tags["TITLE"]).To(Equal([]string{"Synthetic track"}))
		Expect(info.Tags["genre"]).To(Equal(wantGenres))

		after, err := os.Stat(file)
		Expect(err).NotTo(HaveOccurred())
		Expect(os.SameFile(before, after)).To(BeTrue())
		Expect(after.ModTime().Equal(before.ModTime())).To(BeTrue())
		afterTimes, err := times.Stat(file)
		Expect(err).NotTo(HaveOccurred())
		if beforeTimes.HasBirthTime() {
			Expect(afterTimes.HasBirthTime()).To(BeTrue())
			Expect(afterTimes.BirthTime().Equal(beforeTimes.BirthTime())).To(BeTrue())
		}
		written, err := os.ReadFile(file)
		Expect(err).NotTo(HaveOccurred())
		Expect(flacFrames(written)).To(Equal(flacFrames(audio)))

		Expect(lfs.WriteCommentGenres("track.flac", &info)).To(Succeed())
		repeated, err := os.ReadFile(file)
		Expect(err).NotTo(HaveOccurred())
		Expect(repeated).To(Equal(written))
	})

	It("leaves an invalid track untouched if staging fails", func() {
		dir := GinkgoT().TempDir()
		file := filepath.Join(dir, "broken.flac")
		broken := []byte("not a FLAC")
		Expect(os.WriteFile(file, broken, 0600)).To(Succeed())
		stat, err := os.Stat(file)
		Expect(err).NotTo(HaveOccurred())
		lfs := &localFS{root: dir}
		info := metadata.Info{FileInfo: localFileInfo{FileInfo: stat, path: file}, Tags: model.RawTags{"comment": {genreComment}}}
		Expect(lfs.WriteCommentGenres("broken.flac", &info)).NotTo(Succeed())
		after, err := os.ReadFile(file)
		Expect(err).NotTo(HaveOccurred())
		Expect(after).To(Equal(broken))
	})
})

func flacFrames(data []byte) []byte {
	Expect(string(data[:4])).To(Equal("fLaC"))
	for data = data[4:]; ; {
		size := int(binary.BigEndian.Uint32(data[:4]) & 0xffffff)
		last := data[0]&0x80 != 0
		data = data[4+size:]
		if last {
			return data
		}
	}
}
