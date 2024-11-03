package chain

// RunSequentially runs the given functions sequentially,
// If any function returns an error, it stops the execution and returns that error.
// If all functions return nil, it returns nil.
func RunSequentially(fs ...func() error) error {
	for _, f := range fs {
		if err := f(); err != nil {
			return err
		}
	}
	return nil
}
