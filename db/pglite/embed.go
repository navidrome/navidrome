package pglite

import (
	"bytes"
	"compress/gzip"
	_ "embed"
	"fmt"
	"io"
)

// PostgreSQL 17.5 built for WASI (-O2, no wizer snapshot), share files embedded via wasi-vfs.
// build/README.md describes how it is produced. Kept gzipped (4.9 MB vs 14.6 MB); decompressed
// in memory at startup and compiled by wazero. It never touches disk.
//
//go:embed pglite.wasi.gz
var compressedModule []byte

func embeddedModule() ([]byte, error) {
	return gunzip(compressedModule)
}

func gunzip(b []byte) ([]byte, error) {
	zr, err := gzip.NewReader(bytes.NewReader(b))
	if err != nil {
		return nil, fmt.Errorf("opening embedded gzip: %w", err)
	}
	defer zr.Close()
	out, err := io.ReadAll(zr)
	if err != nil {
		return nil, fmt.Errorf("decompressing embedded gzip: %w", err)
	}
	return out, nil
}
