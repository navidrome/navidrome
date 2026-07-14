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

	Describe("ReadManifest", func() {
		It("parses the manifest from a package that also contains wasm", func() {
			ndpPath := filepath.Join(tmpDir, "test.ndp")
			manifest := &Manifest{
				Name:        "Test Plugin",
				Author:      "Test Author",
				Version:     "1.0.0",
				Description: new("A test plugin"),
			}

			err := createTestPackage(ndpPath, manifest, nil)
			Expect(err).ToNot(HaveOccurred())

			m, err := ReadManifest(ndpPath)
			Expect(err).ToNot(HaveOccurred())
			Expect(m.Name).To(Equal("Test Plugin"))
			Expect(*m.Description).To(Equal("A test plugin"))
		})

		It("returns an error for a non-existent file", func() {
			_, err := ReadManifest(filepath.Join(tmpDir, "does-not-exist.ndp"))
			Expect(err).To(HaveOccurred())
		})

		It("returns a specific error for a package missing manifest.json", func() {
			ndpPath := filepath.Join(tmpDir, "no-manifest.ndp")
			f, err := os.Create(ndpPath)
			Expect(err).ToNot(HaveOccurred())
			defer f.Close()
			zw := newTestZipWriter(f)
			Expect(zw.addFile("plugin.wasm", []byte{0x00})).To(Succeed())
			Expect(zw.close()).To(Succeed())

			_, err = ReadManifest(ndpPath)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("missing manifest.json"))
		})

		It("fails for a package with a schema-invalid manifest", func() {
			ndp := filepath.Join(tmpDir, "bad.ndp")
			// empty required fields violate the manifest JSON schema
			err := createTestPackage(ndp, &Manifest{}, nil)
			Expect(err).ToNot(HaveOccurred())
			_, err = ReadManifest(ndp)
			Expect(err).To(HaveOccurred())
		})

		It("enforces cross-field validation", func() {
			ndp := filepath.Join(tmpDir, "crossfield.ndp")
			// subsonicapi permission without users: violates cross-field rule
			manifest := &Manifest{
				Name:        "X",
				Author:      "me",
				Version:     "1.0.0",
				Permissions: &Permissions{Subsonicapi: &SubsonicAPIPermission{}},
			}
			err := createTestPackage(ndp, manifest, nil)
			Expect(err).ToNot(HaveOccurred())

			_, err = ReadManifest(ndp)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("subsonicapi"))
			Expect(err.Error()).To(ContainSubstring("users"))
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
