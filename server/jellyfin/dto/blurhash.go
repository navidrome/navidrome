package dto

import (
	"time"
)

// primaryBlurHash returns the stored blurhash when current for the artwork version, else "" so the
// key is omitted (upstream behavior); clients treat it as cover identity, so absence beats a fake.
func primaryBlurHash(stored string, storedAt *time.Time, version time.Time) string {
	if stored != "" && storedAt != nil && !storedAt.Before(version) {
		return stored
	}
	return ""
}

// primaryBlurHashes builds the ImageBlurHashes map for a known-current hash, or nil so the field is
// omitted entirely when there is none.
func primaryBlurHashes(tag, hash string) map[string]map[string]string {
	if hash == "" {
		return nil
	}
	return map[string]map[string]string{"Primary": {tag: hash}}
}
