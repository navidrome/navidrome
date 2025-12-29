package plugins

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("ndpPackage", func() {
	var tmpDir string

	BeforeEach(func() {
		var err error
		tmpDir, err = os.MkdirTemp("", "plugin-package-test-*")
		Expect(err).ToNot(HaveOccurred())
	})

	AfterEach(func() {
		os.RemoveAll(tmpDir)
	})

	Describe("openPackage", func() {
		It("should load a valid .ndp package", func() {
			ndpPath := filepath.Join(tmpDir, "test.ndp")
			manifest := &Manifest{
				Name:    "Test Plugin",
				Author:  "Test Author",
				Version: "1.0.0",
			}
			wasmBytes := []byte{0x00, 0x61, 0x73, 0x6d} // Minimal wasm header

			err := createTestPackage(ndpPath, manifest, wasmBytes)
			Expect(err).ToNot(HaveOccurred())

			pkg, err := openPackage(ndpPath)
			Expect(err).ToNot(HaveOccurred())
			Expect(pkg.Manifest.Name).To(Equal("Test Plugin"))
			Expect(pkg.Manifest.Author).To(Equal("Test Author"))
			Expect(pkg.Manifest.Version).To(Equal("1.0.0"))
			Expect(pkg.WasmBytes).To(Equal(wasmBytes))
		})

		It("should return error for missing manifest.json", func() {
			ndpPath := filepath.Join(tmpDir, "no-manifest.ndp")

			// Create a zip with only plugin.wasm
			f, err := os.Create(ndpPath)
			Expect(err).ToNot(HaveOccurred())
			defer f.Close()

			zw := newTestZipWriter(f)
			err = zw.addFile("plugin.wasm", []byte{0x00})
			Expect(err).ToNot(HaveOccurred())
			err = zw.close()
			Expect(err).ToNot(HaveOccurred())

			_, err = openPackage(ndpPath)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("missing manifest.json"))
		})

		It("should return error for missing plugin.wasm", func() {
			ndpPath := filepath.Join(tmpDir, "no-wasm.ndp")

			// Create a zip with only manifest.json
			f, err := os.Create(ndpPath)
			Expect(err).ToNot(HaveOccurred())
			defer f.Close()

			zw := newTestZipWriter(f)
			err = zw.addFile("manifest.json", []byte(`{"name":"Test","author":"Test","version":"1.0.0"}`))
			Expect(err).ToNot(HaveOccurred())
			err = zw.close()
			Expect(err).ToNot(HaveOccurred())

			_, err = openPackage(ndpPath)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("missing plugin.wasm"))
		})

		It("should return error for invalid manifest JSON", func() {
			ndpPath := filepath.Join(tmpDir, "invalid-json.ndp")

			f, err := os.Create(ndpPath)
			Expect(err).ToNot(HaveOccurred())
			defer f.Close()

			zw := newTestZipWriter(f)
			err = zw.addFile("manifest.json", []byte(`{invalid json}`))
			Expect(err).ToNot(HaveOccurred())
			err = zw.addFile("plugin.wasm", []byte{0x00})
			Expect(err).ToNot(HaveOccurred())
			err = zw.close()
			Expect(err).ToNot(HaveOccurred())

			_, err = openPackage(ndpPath)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("parsing manifest"))
		})

		It("should return error for manifest missing required fields", func() {
			ndpPath := filepath.Join(tmpDir, "invalid-manifest.ndp")

			f, err := os.Create(ndpPath)
			Expect(err).ToNot(HaveOccurred())
			defer f.Close()

			zw := newTestZipWriter(f)
			err = zw.addFile("manifest.json", []byte(`{"name":"Test"}`)) // Missing author and version
			Expect(err).ToNot(HaveOccurred())
			err = zw.addFile("plugin.wasm", []byte{0x00})
			Expect(err).ToNot(HaveOccurred())
			err = zw.close()
			Expect(err).ToNot(HaveOccurred())

			_, err = openPackage(ndpPath)
			Expect(err).To(HaveOccurred())
			// JSON schema validation happens during unmarshaling
			Expect(err.Error()).To(ContainSubstring("parsing manifest"))
			Expect(err.Error()).To(ContainSubstring("author"))
		})

		It("should return error for non-existent file", func() {
			_, err := openPackage(filepath.Join(tmpDir, "nonexistent.ndp"))
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("opening package"))
		})
	})

	Describe("readManifest", func() {
		It("should read only the manifest without loading wasm", func() {
			ndpPath := filepath.Join(tmpDir, "test.ndp")
			desc := "A test plugin"
			manifest := &Manifest{
				Name:        "Test Plugin",
				Author:      "Test Author",
				Version:     "1.0.0",
				Description: &desc,
			}
			wasmBytes := make([]byte, 1024*1024) // 1MB of zeros

			err := createTestPackage(ndpPath, manifest, wasmBytes)
			Expect(err).ToNot(HaveOccurred())

			m, err := readManifest(ndpPath)
			Expect(err).ToNot(HaveOccurred())
			Expect(m.Name).To(Equal("Test Plugin"))
			Expect(*m.Description).To(Equal("A test plugin"))
		})

		It("should return error for missing manifest", func() {
			ndpPath := filepath.Join(tmpDir, "no-manifest.ndp")

			f, err := os.Create(ndpPath)
			Expect(err).ToNot(HaveOccurred())
			defer f.Close()

			zw := newTestZipWriter(f)
			err = zw.addFile("plugin.wasm", []byte{0x00})
			Expect(err).ToNot(HaveOccurred())
			err = zw.close()
			Expect(err).ToNot(HaveOccurred())

			_, err = readManifest(ndpPath)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("missing manifest.json"))
		})
	})

	Describe("ComputePackageSHA256", func() {
		It("should compute consistent hash for same file", func() {
			ndpPath := filepath.Join(tmpDir, "test.ndp")
			manifest := &Manifest{
				Name:    "Test Plugin",
				Author:  "Test Author",
				Version: "1.0.0",
			}
			wasmBytes := []byte{0x00, 0x61, 0x73, 0x6d}

			err := createTestPackage(ndpPath, manifest, wasmBytes)
			Expect(err).ToNot(HaveOccurred())

			hash1, err := computeFileSHA256(ndpPath)
			Expect(err).ToNot(HaveOccurred())

			hash2, err := computeFileSHA256(ndpPath)
			Expect(err).ToNot(HaveOccurred())

			Expect(hash1).To(Equal(hash2))
			Expect(hash1).To(HaveLen(64)) // SHA-256 produces 64 hex characters
		})
	})
})

// testZipHelper is a helper for creating test zip files with specific contents
type testZipHelper struct {
	f       *os.File
	entries []zipEntry
}

type zipEntry struct {
	name string
	data []byte
}

func newTestZipWriter(f *os.File) *testZipHelper {
	return &testZipHelper{f: f}
}

func (h *testZipHelper) addFile(name string, data []byte) error {
	h.entries = append(h.entries, zipEntry{name: name, data: data})
	return nil
}

func (h *testZipHelper) close() error {
	zw := zip.NewWriter(h.f)
	for _, e := range h.entries {
		w, err := zw.Create(e.name)
		if err != nil {
			return err
		}
		if _, err := w.Write(e.data); err != nil {
			return err
		}
	}
	return zw.Close()
}

// createTestPackage creates an .ndp package file from a manifest and wasm bytes.
// This is primarily used for testing.
func createTestPackage(ndpPath string, manifest *Manifest, wasmBytes []byte) error {
	f, err := os.Create(ndpPath)
	if err != nil {
		return fmt.Errorf("creating package file: %w", err)
	}
	defer f.Close()

	zw := zip.NewWriter(f)
	defer zw.Close()

	// Write manifest.json
	manifestBytes, err := json.Marshal(manifest)
	if err != nil {
		return fmt.Errorf("marshaling manifest: %w", err)
	}

	mw, err := zw.Create(manifestFileName)
	if err != nil {
		return fmt.Errorf("creating manifest in zip: %w", err)
	}
	if _, err := mw.Write(manifestBytes); err != nil {
		return fmt.Errorf("writing manifest: %w", err)
	}

	// Write plugin.wasm
	ww, err := zw.Create(wasmFileName)
	if err != nil {
		return fmt.Errorf("creating wasm in zip: %w", err)
	}
	if _, err := ww.Write(wasmBytes); err != nil {
		return fmt.Errorf("writing wasm: %w", err)
	}

	return nil
}
