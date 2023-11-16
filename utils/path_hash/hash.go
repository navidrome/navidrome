package path_hash

import (
	"crypto/md5"
	"fmt"
)

func PathToMd5Hash(path string) string {
	return fmt.Sprintf("%x", md5.Sum([]byte(path)))
}
