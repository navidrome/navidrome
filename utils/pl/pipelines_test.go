package pl_test

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/navidrome/navidrome/utils/pl"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestPipeline(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Pipeline Tests Suite")
}

var _ = Describe("Pipeline", func() {
	Describe("Stage", func() {
		Context("happy path", func() {
			It("calls the 'transform' function and returns values and errors", func() {
				inC := make(chan int, 4)
				for i := 0; i < 4; i++ {
					inC <- i
				}
				close(inC)

				outC, errC := pl.Stage(context.Background(), 1, inC, func(ctx context.Context, i int) (int, error) {
					if i%2 == 0 {
						return 0, errors.New("even number")
					}
					return i * 2, nil
				})

				Expect(<-errC).To(MatchError("even number"))
				Expect(<-outC).To(Equal(2))
				Expect(<-errC).To(MatchError("even number"))
				Expect(<-outC).To(Equal(6))

				Eventually(outC).Should(BeClosed())
				Eventually(errC).Should(BeClosed())
			})
		})
		Context("Multiple workers", func() {
			const maxWorkers = 2
			const numJobs = 100
			It("starts multiple workers, respecting the limit", func() {
				inC := make(chan int, numJobs)
				for i := 0; i < numJobs; i++ {
					inC <- i
				}
				close(inC)

				current := atomic.Int32{}
				count := atomic.Int32{}
				max := atomic.Int32{}
				outC, _ := pl.Stage(context.Background(), maxWorkers, inC, func(ctx context.Context, in int) (int, error) {
					defer current.Add(-1)
					c := current.Add(1)
					count.Add(1)
					if c > max.Load() {
						max.Store(c)
					}
					time.Sleep(10 * time.Millisecond) // Slow process
					return 0, nil
				})
				// Discard output and wait for completion
				for range outC {
				}

				Expect(count.Load()).To(Equal(int32(numJobs)))
				Expect(current.Load()).To(Equal(int32(0)))
				Expect(max.Load()).To(Equal(int32(maxWorkers)))
			})
		})
		When("the context is canceled", func() {
			It("closes its output", func() {
				ctx, cancel := context.WithCancel(context.Background())
				inC := make(chan int)
				outC, errC := pl.Stage(ctx, 1, inC, func(ctx context.Context, i int) (int, error) {
					return i, nil
				})
				cancel()
				Eventually(outC).Should(BeClosed())
				Eventually(errC).Should(BeClosed())
			})
		})

	})
	Describe("Merge", func() {
		var in1, in2 chan int
		BeforeEach(func() {
			in1 = make(chan int, 4)
			in2 = make(chan int, 4)
			for i := 0; i < 4; i++ {
				in1 <- i
				in2 <- i + 4
			}
			close(in1)
			close(in2)
		})
		When("ranging through the output channel", func() {
			It("copies values from all input channels to its output channel", func() {
				var values []int
				for v := range pl.Merge(context.Background(), in1, in2) {
					values = append(values, v)
				}

				Expect(values).To(ConsistOf(0, 1, 2, 3, 4, 5, 6, 7))
			})
		})
		When("there's a blocked channel and the context is closed", func() {
			It("closes its output", func() {
				ctx, cancel := context.WithCancel(context.Background())
				in3 := make(chan int)
				out := pl.Merge(ctx, in1, in2, in3)
				cancel()
				Eventually(out).Should(BeClosed())
			})
		})
	})
	Describe("ReadOrDone", func() {
		When("values are sent", func() {
			It("copies them to its output channel", func() {
				in := make(chan int)
				out := pl.ReadOrDone(context.Background(), in)
				for i := 0; i < 4; i++ {
					in <- i
					j := <-out
					Expect(i).To(Equal(j))
				}
				close(in)
				Eventually(out).Should(BeClosed())
			})
		})
		When("the context is canceled", func() {
			It("closes its output", func() {
				ctx, cancel := context.WithCancel(context.Background())
				in := make(chan int)
				out := pl.ReadOrDone(ctx, in)
				cancel()
				Eventually(out).Should(BeClosed())
			})
		})
	})
	Describe("SendOrDone", func() {
		When("out is unblocked", func() {
			It("puts the value in the channel", func() {
				out := make(chan int)
				value := 1234
				go pl.SendOrDone(context.Background(), out, value)
				Eventually(out).Should(Receive(&value))
			})
		})
		When("out is blocked", func() {
			It("can be canceled by the context", func() {
				ctx, cancel := context.WithCancel(context.Background())
				out := make(chan int)
				go pl.SendOrDone(ctx, out, 1234)
				cancel()

				Consistently(out).ShouldNot(Receive())
			})
		})
	})
})
