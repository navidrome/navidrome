package playback

import (
	"github.com/navidrome/navidrome/model"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Queues", func() {
	var queue *Queue

	BeforeEach(func() {
		queue = NewQueue()
	})

	Describe("use empty queue", func() {
		It("is empty", func() {
			Expect(queue.Items).To(BeEmpty())
			Expect(queue.Index).To(Equal(-1))
		})
	})

	Describe("Operate on small queue", func() {
		BeforeEach(func() {
			mfs := model.MediaFiles{
				{
					ID: "1", Artist: "Queen", Compilation: false, Path: "/music1/hammer.mp3",
				},
				{
					ID: "2", Artist: "Vinyard Rose", Compilation: false, Path: "/music1/cassidy.mp3",
				},
			}
			queue.Add(mfs)
		})

		It("contains the preloaded data", func() {
			Expect(queue.Get).ToNot(BeNil())
			Expect(queue.Size()).To(Equal(2))
		})

		It("could read data by ID", func() {
			idx, err := queue.getMediaFileIndexByID("1")
			Expect(err).ToNot(HaveOccurred())
			Expect(idx).ToNot(BeNil())
			Expect(idx).To(Equal(0))

			queue.SetIndex(idx)

			mf := queue.Current()

			Expect(mf).ToNot(BeNil())
			Expect(mf.ID).To(Equal("1"))
			Expect(mf.Artist).To(Equal("Queen"))
			Expect(mf.Path).To(Equal("/music1/hammer.mp3"))
		})
	})

	Describe("Read/Write operations", func() {
		BeforeEach(func() {
			mfs := model.MediaFiles{
				{
					ID: "1", Artist: "Queen", Compilation: false, Path: "/music1/hammer.mp3",
				},
				{
					ID: "2", Artist: "Vinyard Rose", Compilation: false, Path: "/music1/cassidy.mp3",
				},
				{
					ID: "3", Artist: "Pink Floyd", Compilation: false, Path: "/music1/time.mp3",
				},
				{
					ID: "4", Artist: "Mike Oldfield", Compilation: false, Path: "/music1/moonlight-shadow.mp3",
				},
				{
					ID: "5", Artist: "Red Hot Chili Peppers", Compilation: false, Path: "/music1/californication.mp3",
				},
			}
			queue.Add(mfs)
		})

		It("contains the preloaded data", func() {
			Expect(queue.Get).ToNot(BeNil())
			Expect(queue.Size()).To(Equal(5))
		})

		It("could read data by ID", func() {
			idx, err := queue.getMediaFileIndexByID("5")
			Expect(err).ToNot(HaveOccurred())
			Expect(idx).ToNot(BeNil())
			Expect(idx).To(Equal(4))

			queue.SetIndex(idx)

			mf := queue.Current()

			Expect(mf).ToNot(BeNil())
			Expect(mf.ID).To(Equal("5"))
			Expect(mf.Artist).To(Equal("Red Hot Chili Peppers"))
			Expect(mf.Path).To(Equal("/music1/californication.mp3"))
		})

		It("could shuffle the data correctly", func() {
			queue.Shuffle()
			Expect(queue.Size()).To(Equal(5))
		})

		It("could remove entries correctly", func() {
			queue.Remove(0)
			Expect(queue.Size()).To(Equal(4))

			queue.Remove(3)
			Expect(queue.Size()).To(Equal(3))
		})

		It("clear the whole thing on request", func() {
			Expect(queue.Size()).To(Equal(5))
			queue.Clear()
			Expect(queue.Size()).To(Equal(0))
		})
	})

})
