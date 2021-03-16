//+build !cgo

package metadata

import (
	"fmt"
)

type taglibExtractor struct{}

func (e *taglibExtractor) Extract(paths ...string) (map[string]Metadata, error) {
	return nil, fmt.Errorf("compiled without CGO")
}
