package engine

import (
	"sync"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("NowPlayingRepository", func() {
	var repo NowPlayingRepository
	var now = time.Now()
	var past = time.Time{}

	BeforeEach(func() {
		playerMap = sync.Map{}
		repo = NewNowPlayingRepository()
	})

	It("enqueues and dequeues records", func() {
		Expect(repo.Enqueue(&NowPlayingInfo{PlayerId: 1, TrackID: "AAA", Start: now})).To(BeNil())
		Expect(repo.Enqueue(&NowPlayingInfo{PlayerId: 1, TrackID: "BBB", Start: now})).To(BeNil())

		Expect(repo.Tail(1)).To(Equal(&NowPlayingInfo{PlayerId: 1, TrackID: "AAA", Start: now}))
		Expect(repo.Head(1)).To(Equal(&NowPlayingInfo{PlayerId: 1, TrackID: "BBB", Start: now}))

		Expect(repo.Count(1)).To(Equal(int64(2)))

		Expect(repo.Dequeue(1)).To(Equal(&NowPlayingInfo{PlayerId: 1, TrackID: "AAA", Start: now}))
		Expect(repo.Count(1)).To(Equal(int64(1)))
	})

	It("handles multiple players", func() {
		Expect(repo.Enqueue(&NowPlayingInfo{PlayerId: 1, TrackID: "AAA", Start: now})).To(BeNil())
		Expect(repo.Enqueue(&NowPlayingInfo{PlayerId: 1, TrackID: "BBB", Start: now})).To(BeNil())

		Expect(repo.Enqueue(&NowPlayingInfo{PlayerId: 2, TrackID: "CCC", Start: now})).To(BeNil())
		Expect(repo.Enqueue(&NowPlayingInfo{PlayerId: 2, TrackID: "DDD", Start: now})).To(BeNil())

		Expect(repo.GetAll()).To(ConsistOf([]*NowPlayingInfo{
			{PlayerId: 1, TrackID: "BBB", Start: now},
			{PlayerId: 2, TrackID: "DDD", Start: now},
		}))

		Expect(repo.Count(2)).To(Equal(int64(2)))
		Expect(repo.Count(2)).To(Equal(int64(2)))

		Expect(repo.Tail(1)).To(Equal(&NowPlayingInfo{PlayerId: 1, TrackID: "AAA", Start: now}))
		Expect(repo.Head(2)).To(Equal(&NowPlayingInfo{PlayerId: 2, TrackID: "DDD", Start: now}))
	})

	It("handles expired items", func() {
		Expect(repo.Enqueue(&NowPlayingInfo{PlayerId: 1, TrackID: "AAA", Start: past})).To(BeNil())
		Expect(repo.Enqueue(&NowPlayingInfo{PlayerId: 2, TrackID: "BBB", Start: now})).To(BeNil())

		Expect(repo.GetAll()).To(ConsistOf([]*NowPlayingInfo{
			{PlayerId: 2, TrackID: "BBB", Start: now},
		}))
	})
})
