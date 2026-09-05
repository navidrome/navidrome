package pglite

import (
	"crypto/rand"
	"fmt"
	"os"
	"path/filepath"
)

// setupDataDir prepares the host directories the guest mounts (cluster, /dev/urandom stand-in,
// initdb password file) and reports whether initdb still has to run.
func setupDataDir(dataDir string) (fresh bool, err error) {
	for _, dir := range []string{"pgdata", "dev", "pglite"} {
		if err := os.MkdirAll(filepath.Join(dataDir, dir), 0o700); err != nil {
			return false, fmt.Errorf("creating %s: %w", dir, err)
		}
	}
	urandom := filepath.Join(dataDir, "dev", "urandom")
	if !fileExists(urandom) {
		seed := make([]byte, 256)
		if _, err := rand.Read(seed); err != nil {
			return false, fmt.Errorf("generating random bytes: %w", err)
		}
		if err := os.WriteFile(urandom, seed, 0o600); err != nil {
			return false, fmt.Errorf("writing urandom: %w", err)
		}
	}
	// initdb's --pwfile; the bridge never checks it.
	if err := os.WriteFile(filepath.Join(dataDir, "pglite", "password"), []byte("password\n"), 0o600); err != nil {
		return false, fmt.Errorf("writing password file: %w", err)
	}
	return !fileExists(filepath.Join(dataDir, "pgdata", "PG_VERSION")), nil
}
