package utils

import (
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/navidrome/navidrome/model/id"
)

var cleanFileNameRe = regexp.MustCompile(`[^a-z0-9_-]`)

func TempFileName(prefix, suffix string) string {
	return filepath.Join(os.TempDir(), prefix+id.NewRandom()+suffix)
}

func BaseName(filePath string) string {
	p := path.Base(filePath)
	return strings.TrimSuffix(p, path.Ext(p))
}

// CleanFileName produces a filesystem-safe, human-readable version of a name.
// It lowercases, replaces spaces with underscores, strips non-alphanumeric
// characters (except underscore and hyphen), and truncates to 50 characters.
func CleanFileName(name string) string {
	s := strings.ToLower(strings.TrimSpace(name))
	s = strings.ReplaceAll(s, " ", "_")
	s = cleanFileNameRe.ReplaceAllString(s, "")
	if len(s) > 50 {
		s = s[:50]
	}
	s = strings.TrimRight(s, "_-")
	return s
}

// FileExists checks if a file or directory exists
func FileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil || !os.IsNotExist(err)
}

// RelPath is filepath.Rel without the UNC share root hang (golang/go#79784). Remove it after moving to Go 1.28.
func RelPath(basePath, targPath string) (string, error) {
	return filepath.Rel(normalizeUNCRoot(basePath), normalizeUNCRoot(targPath))
}

func normalizeUNCRoot(p string) string {
	vol := filepath.VolumeName(p)
	if len(vol) > 2 && strings.Trim(p[len(vol):], `\/`) == "" {
		return vol + string(filepath.Separator)
	}
	return p
}
