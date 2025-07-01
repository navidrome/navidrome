package run

import "golang.org/x/sync/errgroup"

// Sequentially runs the given functions sequentially,
// If any function returns an error, it stops the execution and returns that error.
// If all functions return nil, it returns nil.
func Sequentially(fs ...func() error) error {
	for _, f := range fs {
		if err := f(); err != nil {
			return err
		}
	}
	return nil
}

// Parallel runs the given functions in parallel,
// It waits for all functions to finish and returns the first error encountered.
func Parallel(fs ...func() error) func() error {
	return func() error {
		g := errgroup.Group{}
		for _, f := range fs {
			g.Go(func() error {
				return f()
			})
		}
		return g.Wait()
	}
}
