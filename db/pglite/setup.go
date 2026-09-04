package pglite

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/rand"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// WASIBinary holds the pglite-wasi tar.gz contents, read from Config.Tarball on first use.
var WASIBinary []byte

// setupEnvironment extracts the archive on first run and returns the pglite.wasi bytes.
func setupEnvironment(dataDir string) (wasmBinary []byte, err error) {
	pgBaseDir := filepath.Join(dataDir, "pglite")

	versionFile := filepath.Join(pgBaseDir, "base", "PG_VERSION")
	wasmFile := filepath.Join(pgBaseDir, "bin", "pglite.wasi")
	if _, err := os.Stat(versionFile); err != nil && !fileExists(wasmFile) {
		if WASIBinary == nil {
			return nil, fmt.Errorf("pglite WASI binary not available (WASIBinary is nil)")
		}
		if err := extractTarGz(dataDir, WASIBinary); err != nil {
			return nil, fmt.Errorf("extracting pglite-wasi.tar.gz: %w", err)
		}
	}

	devDir := filepath.Join(dataDir, "dev")
	if err := os.MkdirAll(devDir, 0o755); err != nil {
		return nil, fmt.Errorf("creating dev dir: %w", err)
	}
	urandomPath := filepath.Join(devDir, "urandom")
	if _, err := os.Stat(urandomPath); err != nil {
		randomBytes := make([]byte, 256)
		if _, err := rand.Read(randomBytes); err != nil {
			return nil, fmt.Errorf("generating random bytes: %w", err)
		}
		if err := os.WriteFile(urandomPath, randomBytes, 0o644); err != nil {
			return nil, fmt.Errorf("writing urandom: %w", err)
		}
	}

	wasmPath := filepath.Join(pgBaseDir, "bin", "pglite.wasi")
	wasmBinary, err = os.ReadFile(wasmPath)
	if err != nil {
		return nil, fmt.Errorf("reading pglite.wasi: %w", err)
	}

	return wasmBinary, nil
}

func extractTarGz(destDir string, data []byte) error {
	gzReader, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("opening gzip: %w", err)
	}
	defer gzReader.Close()

	tarReader := tar.NewReader(gzReader)

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("reading tar entry: %w", err)
		}

		// destDir is mounted as /tmp in the guest, so drop the archive's leading "tmp/".
		name := header.Name
		name = strings.TrimPrefix(name, "tmp/")
		if name == "" {
			continue
		}

		target := filepath.Join(destDir, name)

		if !strings.HasPrefix(filepath.Clean(target), filepath.Clean(destDir)) {
			return fmt.Errorf("tar entry %q escapes destination", header.Name)
		}

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, os.FileMode(header.Mode)); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return err
			}
			f, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(header.Mode))
			if err != nil {
				return err
			}
			if _, err := io.Copy(f, tarReader); err != nil {
				f.Close()
				return err
			}
			f.Close()
		case tar.TypeSymlink:
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return err
			}
			os.Remove(target)
			if err := os.Symlink(header.Linkname, target); err != nil {
				return err
			}
		}
	}

	return nil
}
