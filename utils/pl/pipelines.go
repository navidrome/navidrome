// Package pl implements some Data Pipeline helper functions.
// Reference: https://medium.com/amboss/applying-modern-go-concurrency-patterns-to-data-pipelines-b3b5327908d4#3a80
//
// See also:
//
//	https://www.oreilly.com/library/view/concurrency-in-go/9781491941294/ch04.html#fano_fani
//	https://www.youtube.com/watch?v=f6kdp27TYZs
//	https://www.youtube.com/watch?v=QDDwwePbDtw
package pl

import (
	"context"
	"errors"
	"sync"

	"github.com/navidrome/navidrome/log"
	"golang.org/x/sync/semaphore"
)

func Stage[In any, Out any](
	ctx context.Context,
	maxWorkers int,
	inputChannel <-chan In,
	fn func(context.Context, In) (Out, error),
) (chan Out, chan error) {
	outputChannel := make(chan Out)
	errorChannel := make(chan error)

	limit := int64(maxWorkers)
	sem1 := semaphore.NewWeighted(limit)

	go func() {
		defer close(outputChannel)
		defer close(errorChannel)

		for s := range ReadOrDone(ctx, inputChannel) {
			if err := sem1.Acquire(ctx, 1); err != nil {
				if !errors.Is(err, context.Canceled) {
					log.Error(ctx, "Failed to acquire semaphore", err)
				}
				break
			}

			go func(s In) {
				defer sem1.Release(1)

				result, err := fn(ctx, s)
				if err != nil {
					if !errors.Is(err, context.Canceled) {
						errorChannel <- err
					}
				} else {
					outputChannel <- result
				}
			}(s)
		}

		// By using context.Background() here we are assuming the fn will stop when the context
		// is canceled. This is required so we can wait for the workers to finish and avoid closing
		// the outputChannel before they are done.
		if err := sem1.Acquire(context.Background(), limit); err != nil {
			log.Error(ctx, "Failed waiting for workers", err)
		}
	}()

	return outputChannel, errorChannel
}

func Sink[In any](
	ctx context.Context,
	maxWorkers int,
	inputChannel <-chan In,
	fn func(context.Context, In) error,
) chan error {
	results, errC := Stage(ctx, maxWorkers, inputChannel, func(ctx context.Context, in In) (bool, error) {
		err := fn(ctx, in)
		return false, err // Only err is important, results will be discarded
	})

	// Discard results
	go func() {
		for range ReadOrDone(ctx, results) {
		}
	}()

	return errC
}

func Merge[T any](ctx context.Context, cs ...<-chan T) <-chan T {
	var wg sync.WaitGroup
	out := make(chan T)

	output := func(c <-chan T) {
		defer wg.Done()
		for v := range ReadOrDone(ctx, c) {
			select {
			case out <- v:
			case <-ctx.Done():
				return
			}
		}
	}

	wg.Add(len(cs))
	for _, c := range cs {
		go output(c)
	}

	go func() {
		wg.Wait()
		close(out)
	}()

	return out
}

func SendOrDone[T any](ctx context.Context, out chan<- T, v T) {
	select {
	case out <- v:
	case <-ctx.Done():
		return
	}
}

func ReadOrDone[T any](ctx context.Context, in <-chan T) <-chan T {
	valStream := make(chan T)
	go func() {
		defer close(valStream)
		for {
			select {
			case <-ctx.Done():
				return
			case v, ok := <-in:
				if !ok {
					return
				}
				select {
				case valStream <- v:
				case <-ctx.Done():
				}
			}
		}
	}()
	return valStream
}

func Tee[T any](ctx context.Context, in <-chan T) (<-chan T, <-chan T) {
	out1 := make(chan T)
	out2 := make(chan T)
	go func() {
		defer close(out1)
		defer close(out2)
		for val := range ReadOrDone(ctx, in) {
			var out1, out2 = out1, out2
			for i := 0; i < 2; i++ {
				select {
				case <-ctx.Done():
				case out1 <- val:
					out1 = nil
				case out2 <- val:
					out2 = nil
				}
			}
		}
	}()
	return out1, out2
}

func FromSlice[T any](ctx context.Context, in []T) <-chan T {
	output := make(chan T, len(in))
	for _, c := range in {
		output <- c
	}
	close(output)
	return output
}
