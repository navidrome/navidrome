package pool

import (
	"testing"

	"github.com/deluan/navidrome/log"
	"github.com/deluan/navidrome/tests"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestCore(t *testing.T) {
	tests.Init(t, false)
	log.SetLevel(log.LevelCritical)
	RegisterFailHandler(Fail)
	RunSpecs(t, "Core Suite")
}

type testItem struct {
	ID int
}

type results []int

func (r results) Len() int { return len(r) }

var processed results

var _ = Describe("Pool", func() {
	var pool *Pool

	BeforeEach(func() {
		processed = nil
		pool, _ = NewPool("test", 2, &testItem{}, execute)
	})

	It("processes items", func() {
		for i := 0; i < 5; i++ {
			pool.Submit(&testItem{ID: i})
		}
		Eventually(processed.Len, "10s").Should(Equal(5))
		Expect(processed).To(ContainElements(0, 1, 2, 3, 4))
	})
})

func execute(workload interface{}) {
	item := workload.(*testItem)
	processed = append(processed, item.ID)
}
