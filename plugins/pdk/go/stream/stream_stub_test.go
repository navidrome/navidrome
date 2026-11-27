//go:build !wasip1

package stream

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStubStreamFailsOutsideWasm(t *testing.T) {
	s := New(7)
	assert.Equal(t, uint32(7), s.Handle())

	_, err := s.Read(make([]byte, 1))
	assert.ErrorIs(t, err, errNotWasm)
	_, err = s.Write([]byte("x"))
	assert.ErrorIs(t, err, errNotWasm)
	assert.ErrorIs(t, s.Close(), errNotWasm)
	_, err = Copy(s, New(8))
	assert.ErrorIs(t, err, errNotWasm)
}
