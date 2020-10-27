package core

import (
	"sync"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("NowPlaying", func() {
	var repo *nowPlayingRepository
	var now = time.Now()
	var past = time.Time{}

	BeforeEach(func() {
		playerMap = sync.Map{}
		repo = NewNowPlayingRepository().(*nowPlayingRepository)
	})

	It("enqueues and dequeues records", func() {
		Expect(repo.Enqueue(&NowPlayingInfo{PlayerId: 1, TrackID: "AAA", Start: now})).To(BeNil())
		Expect(repo.Enqueue(&NowPlayingInfo{PlayerId: 1, TrackID: "BBB", Start: now})).To(BeNil())

		Expect(repo.tail(1)).To(Equal(&NowPlayingInfo{PlayerId: 1, TrackID: "AAA", Start: now}))
		Expect(repo.head(1)).To(Equal(&NowPlayingInfo{PlayerId: 1, TrackID: "BBB", Start: now}))

		Expect(repo.count(1)).To(Equal(int64(2)))

		Expect(repo.dequeue(1)).To(Equal(&NowPlayingInfo{PlayerId: 1, TrackID: "AAA", Start: now}))
		Expect(repo.count(1)).To(Equal(int64(1)))
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

		Expect(repo.count(2)).To(Equal(int64(2)))
		Expect(repo.count(2)).To(Equal(int64(2)))

		Expect(repo.tail(1)).To(Equal(&NowPlayingInfo{PlayerId: 1, TrackID: "AAA", Start: now}))
		Expect(repo.head(2)).To(Equal(&NowPlayingInfo{PlayerId: 2, TrackID: "DDD", Start: now}))
	})

	It("handles expired items", func() {
		Expect(repo.Enqueue(&NowPlayingInfo{PlayerId: 1, TrackID: "AAA", Start: past})).To(BeNil())
		Expect(repo.Enqueue(&NowPlayingInfo{PlayerId: 2, TrackID: "BBB", Start: now})).To(BeNil())

		Expect(repo.GetAll()).To(ConsistOf([]*NowPlayingInfo{
			{PlayerId: 2, TrackID: "BBB", Start: now},
		}))
	})
})
