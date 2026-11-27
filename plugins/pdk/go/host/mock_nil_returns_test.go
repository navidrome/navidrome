//go:build !wasip1

package host

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// Return(nil, ...) models "no result"; the generated accessors must yield the
// zero value instead of panicking on the type assertion.
func TestMockNilPointerReturn(t *testing.T) {
	HTTPMock.On("Send", HTTPRequest{URL: "http://nil.example"}).Return(nil, nil)

	resp, err := HTTPSend(HTTPRequest{URL: "http://nil.example"})
	require.NoError(t, err)
	require.Nil(t, resp)
}

func TestMockNilSliceReturn(t *testing.T) {
	LibraryMock.On("GetAllLibraries").Return(nil, nil)

	libs, err := LibraryGetAllLibraries()
	require.NoError(t, err)
	require.Nil(t, libs)
}
