//nolint:unused
package storagetest_test

import (
	"testing"
	"testing/fstest"
	"time"

	. "github.com/navidrome/navidrome/core/storage/storagetest"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

type _t = map[string]any

func TestFakeStorage(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Fake Storage Test Suite")
}

var _ = Describe("FakeFS", func() {
	var ffs FakeFS
	var startTime time.Time

	BeforeEach(func() {
		startTime = time.Now().Add(-time.Hour)
		boy := Template(_t{"albumartist": "U2", "album": "Boy", "year": 1980, "genre": "Rock"})
		files := fstest.MapFS{
			"U2/Boy/I Will Follow.mp3": boy(Track(1, "I Will Follow")),
			"U2/Boy/Twilight.mp3":      boy(Track(2, "Twilight")),
			"U2/Boy/An Cat Dubh.mp3":   boy(Track(3, "An Cat Dubh")),
		}
		ffs.SetFiles(files)
	})

	It("should implement a fs.FS", func() {
		Expect(fstest.TestFS(ffs, "U2/Boy/I Will Follow.mp3")).To(Succeed())
	})

	It("should read file info", func() {
		props, err := ffs.ReadTags("U2/Boy/I Will Follow.mp3", "U2/Boy/Twilight.mp3")
		Expect(err).ToNot(HaveOccurred())

		prop := props["U2/Boy/Twilight.mp3"]
		Expect(prop).ToNot(BeNil())
		Expect(prop.AudioProperties.Channels).To(Equal(2))
		Expect(prop.AudioProperties.BitRate).To(Equal(320))
		Expect(prop.FileInfo.Name()).To(Equal("Twilight.mp3"))
		Expect(prop.Tags["albumartist"]).To(ConsistOf("U2"))
		Expect(prop.FileInfo.ModTime()).To(BeTemporally(">=", startTime))

		prop = props["U2/Boy/I Will Follow.mp3"]
		Expect(prop).ToNot(BeNil())
		Expect(prop.FileInfo.Name()).To(Equal("I Will Follow.mp3"))
	})

	It("should return ModTime for directories", func() {
		root := ffs.MapFS["."]
		dirInfo1, err := ffs.Stat("U2")
		dirInfo2, err := ffs.Stat("U2/Boy")
		Expect(err).ToNot(HaveOccurred())
		Expect(dirInfo1.ModTime()).To(Equal(root.ModTime))
		Expect(dirInfo1.ModTime()).To(BeTemporally(">=", startTime))
		Expect(dirInfo1.ModTime()).To(Equal(dirInfo2.ModTime()))
	})

	It("should only update the file's directory ModTime when the file is touched", func() {
		root, _ := ffs.Stat(".")
		dirInfo1, _ := ffs.Stat("U2")
		dirInfo2, _ := ffs.Stat("U2/Boy")
		previousTime := root.ModTime()

		aTimeStamp := previousTime.Add(time.Hour)
		ffs.Touch("U2/Boy/Twilight.mp3", aTimeStamp)

		Expect(root.ModTime()).To(Equal(previousTime))
		Expect(dirInfo1.ModTime()).To(Equal(previousTime))
		Expect(dirInfo2.ModTime()).To(Equal(aTimeStamp))
	})
})
