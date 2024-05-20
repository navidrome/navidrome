package metadata

import (
	"crypto/md5"
	"fmt"
)

func (md Metadata) trackID() string {
	return fmt.Sprintf("%x", md5.Sum([]byte(md.FilePath())))
}
