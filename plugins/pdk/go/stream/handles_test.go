package stream

import (
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestReaderAndWriterMapZeroToNil(t *testing.T) {
	assert.Nil(t, Reader(0))
	assert.Nil(t, Writer(0))
	assert.Equal(t, uint32(5), Reader(5).(*Stream).Handle())
	assert.Equal(t, uint32(6), Writer(6).(*Stream).Handle())
}

func TestHandleOf(t *testing.T) {
	h, err := HandleOf(nil)
	assert.NoError(t, err)
	assert.Zero(t, h)

	var none *Stream
	h, err = HandleOf(none)
	assert.NoError(t, err)
	assert.Zero(t, h)

	var rc io.ReadCloser = New(9)
	h, err = HandleOf(rc)
	assert.NoError(t, err)
	assert.Equal(t, uint32(9), h)

	_, err = HandleOf(io.NopCloser(strings.NewReader("mine")))
	assert.EqualError(t, err, "only streams from Navidrome can be passed to a host function")
}
