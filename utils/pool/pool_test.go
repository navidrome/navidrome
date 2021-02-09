package pool

import (
	"testing"

	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestPool(t *testing.T) {
	tests.Init(t, false)
	log.SetLevel(log.LevelCritical)
	RegisterFailHandler(Fail)
	RunSpecs(t, "Pool Suite")
}

type testItem struct {
	ID int
}

var processed []int

var _ = Describe("Pool", func() {
	var pool *Pool

	BeforeEach(func() {
		processed = nil
		pool, _ = NewPool("test", 2, execute)
	})

	It("processes items", func() {
		for i := 0; i < 5; i++ {
			pool.Submit(&testItem{ID: i})
		}
		Eventually(func() []int { return processed }, "10s").Should(HaveLen(5))
		Expect(processed).To(ContainElements(0, 1, 2, 3, 4))
	})
})

func execute(workload interface{}) {
	item := workload.(*testItem)
	processed = append(processed, item.ID)
}
