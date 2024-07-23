package gravatar

import (
	"crypto/sha256"
	"fmt"
	"strings"
)

const baseUrl = "https://www.gravatar.com/avatar"
const defaultSize = 80
const maxSize = 2048

func Url(email string, size int) string {
	email = strings.ToLower(email)
	email = strings.TrimSpace(email)
	hash := sha256.Sum256([]byte(email))
	if size < 1 {
		size = defaultSize
	}
	size = min(maxSize, size)

	return fmt.Sprintf("%s/%x?s=%d", baseUrl, hash, size)
}
