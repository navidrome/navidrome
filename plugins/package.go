package plugins

import (
	"archive/zip"
	"errors"
	"fmt"
	"io"
)

const (
	// PackageExtension is the file extension for Navidrome plugin packages.
	PackageExtension = ".ndp"

	// manifestFileName is the name of the manifest file inside the package.
	manifestFileName = "manifest.json"

	// wasmFileName is the name of the WebAssembly module inside the package.
	wasmFileName = "plugin.wasm"
)

// ndpPackage represents a loaded .ndp plugin package.
// It contains the manifest and wasm bytes read from the archive.
type ndpPackage struct {
	Manifest  *Manifest
	WasmBytes []byte
}

// openPackage opens an .ndp file and extracts the manifest and wasm bytes.
// The caller does not need to call Close() - all resources are read into memory.
func openPackage(ndpPath string) (*ndpPackage, error) {
	// Open the zip archive
	zr, err := zip.OpenReader(ndpPath)
	if err != nil {
		return nil, fmt.Errorf("opening package: %w", err)
	}
	defer zr.Close()

	var manifestBytes []byte
	var wasmBytes []byte

	for _, f := range zr.File {
		switch f.Name {
		case manifestFileName:
			manifestBytes, err = readZipFile(f)
			if err != nil {
				return nil, fmt.Errorf("reading manifest: %w", err)
			}
		case wasmFileName:
			wasmBytes, err = readZipFile(f)
			if err != nil {
				return nil, fmt.Errorf("reading wasm: %w", err)
			}
		}
	}

	if manifestBytes == nil {
		return nil, errors.New("package missing manifest.json")
	}
	if wasmBytes == nil {
		return nil, errors.New("package missing plugin.wasm")
	}

	// Parse and validate manifest
	manifest, err := ParseManifest(manifestBytes)
	if err != nil {
		return nil, fmt.Errorf("parsing manifest: %w", err)
	}

	return &ndpPackage{
		Manifest:  manifest,
		WasmBytes: wasmBytes,
	}, nil
}

// readManifest reads only the manifest from an .ndp file without loading the wasm bytes.
// This is useful for quick plugin discovery.
func readManifest(ndpPath string) (*Manifest, error) {
	// Open the zip archive
	zr, err := zip.OpenReader(ndpPath)
	if err != nil {
		return nil, fmt.Errorf("opening package: %w", err)
	}
	defer zr.Close()

	for _, f := range zr.File {
		if f.Name == manifestFileName {
			manifestBytes, err := readZipFile(f)
			if err != nil {
				return nil, fmt.Errorf("reading manifest: %w", err)
			}

			manifest, err := ParseManifest(manifestBytes)
			if err != nil {
				return nil, fmt.Errorf("parsing manifest: %w", err)
			}

			return manifest, nil
		}
	}

	return nil, errors.New("package missing manifest.json")
}

// readZipFile reads the contents of a file from a zip archive.
func readZipFile(f *zip.File) ([]byte, error) {
	rc, err := f.Open()
	if err != nil {
		return nil, err
	}
	defer rc.Close()
	return io.ReadAll(rc)
}
