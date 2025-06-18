package plugins

import (
	"archive/zip"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/navidrome/navidrome/plugins/schema"
)

// PluginPackage represents a Navidrome Plugin Package (.ndp file)
type PluginPackage struct {
	ManifestJSON []byte
	Manifest     *schema.PluginManifest
	WasmBytes    []byte
	Docs         map[string][]byte
}

// ExtractPackage extracts a .ndp file to the target directory
func ExtractPackage(ndpPath, targetDir string) error {
	r, err := zip.OpenReader(ndpPath)
	if err != nil {
		return fmt.Errorf("error opening .ndp file: %w", err)
	}
	defer r.Close()

	// Create target directory if it doesn't exist
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return fmt.Errorf("error creating plugin directory: %w", err)
	}

	// Define a reasonable size limit for plugin files to prevent decompression bombs
	const maxFileSize = 10 * 1024 * 1024 // 10 MB limit

	// Extract all files from the zip
	for _, f := range r.File {
		// Skip directories (they will be created as needed)
		if f.FileInfo().IsDir() {
			continue
		}

		// Create the file path for extraction
		// Validate the file name to prevent directory traversal or absolute paths
		if strings.Contains(f.Name, "..") || filepath.IsAbs(f.Name) {
			return fmt.Errorf("illegal file path in plugin package: %s", f.Name)
		}

		// Create the file path for extraction
		targetPath := filepath.Join(targetDir, f.Name) // #nosec G305

		// Clean the path to prevent directory traversal.
		cleanedPath := filepath.Clean(targetPath)
		// Ensure the cleaned path is still within the target directory.
		// We resolve both paths to absolute paths to be sure.
		absTargetDir, err := filepath.Abs(targetDir)
		if err != nil {
			return fmt.Errorf("failed to resolve target directory path: %w", err)
		}
		absTargetPath, err := filepath.Abs(cleanedPath)
		if err != nil {
			return fmt.Errorf("failed to resolve extracted file path: %w", err)
		}
		if !strings.HasPrefix(absTargetPath, absTargetDir+string(os.PathSeparator)) && absTargetPath != absTargetDir {
			return fmt.Errorf("illegal file path in plugin package: %s", f.Name)
		}

		// Open the file inside the zip
		rc, err := f.Open()
		if err != nil {
			return fmt.Errorf("error opening file in plugin package: %w", err)
		}

		// Create parent directories if they don't exist
		if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
			rc.Close()
			return fmt.Errorf("error creating directory structure: %w", err)
		}

		// Create the file
		outFile, err := os.Create(targetPath)
		if err != nil {
			rc.Close()
			return fmt.Errorf("error creating extracted file: %w", err)
		}

		// Copy the file contents with size limit
		if _, err := io.CopyN(outFile, rc, maxFileSize); err != nil && !errors.Is(err, io.EOF) {
			outFile.Close()
			rc.Close()
			if errors.Is(err, io.ErrUnexpectedEOF) { // File size exceeds limit
				return fmt.Errorf("error extracting file: size exceeds limit (%d bytes) for %s", maxFileSize, f.Name)
			}
			return fmt.Errorf("error writing extracted file: %w", err)
		}

		outFile.Close()
		rc.Close()

		// Set appropriate file permissions (0600 - readable only by owner)
		if err := os.Chmod(targetPath, 0600); err != nil {
			return fmt.Errorf("error setting permissions on extracted file: %w", err)
		}
	}

	return nil
}

// LoadPackage loads and validates an .ndp file without extracting it
func LoadPackage(ndpPath string) (*PluginPackage, error) {
	r, err := zip.OpenReader(ndpPath)
	if err != nil {
		return nil, fmt.Errorf("error opening .ndp file: %w", err)
	}
	defer r.Close()

	pkg := &PluginPackage{
		Docs: make(map[string][]byte),
	}

	// Required files
	var hasManifest, hasWasm bool

	// Read all files in the zip
	for _, f := range r.File {
		// Skip directories
		if f.FileInfo().IsDir() {
			continue
		}

		// Get file content
		rc, err := f.Open()
		if err != nil {
			return nil, fmt.Errorf("error opening file in plugin package: %w", err)
		}

		content, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			return nil, fmt.Errorf("error reading file in plugin package: %w", err)
		}

		// Process based on file name
		switch strings.ToLower(f.Name) {
		case "manifest.json":
			pkg.ManifestJSON = content
			hasManifest = true
		case "plugin.wasm":
			pkg.WasmBytes = content
			hasWasm = true
		default:
			// Store other files as documentation
			pkg.Docs[f.Name] = content
		}
	}

	// Ensure required files exist
	if !hasManifest {
		return nil, fmt.Errorf("plugin package missing required manifest.json")
	}
	if !hasWasm {
		return nil, fmt.Errorf("plugin package missing required plugin.wasm")
	}

	// Parse and validate the manifest
	var manifest schema.PluginManifest
	if err := json.Unmarshal(pkg.ManifestJSON, &manifest); err != nil {
		return nil, fmt.Errorf("invalid manifest: %w", err)
	}

	pkg.Manifest = &manifest
	return pkg, nil
}
