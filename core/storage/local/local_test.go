package local

import (
	"io/fs"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/conf/configtest"
	"github.com/navidrome/navidrome/core/storage"
	"github.com/navidrome/navidrome/model/metadata"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("LocalStorage", func() {
	var tempDir string
	var testExtractor *mockTestExtractor

	BeforeEach(func() {
		DeferCleanup(configtest.SetupConfig())

		// Create a temporary directory for testing
		var err error
		tempDir, err = os.MkdirTemp("", "navidrome-local-storage-test-")
		Expect(err).ToNot(HaveOccurred())

		DeferCleanup(func() {
			os.RemoveAll(tempDir)
		})

		// Create and register a test extractor
		testExtractor = &mockTestExtractor{
			results: make(map[string]metadata.Info),
		}
		RegisterExtractor("test", func(fs.FS, string) Extractor {
			return testExtractor
		})
		conf.Server.Scanner.Extractor = "test"
	})

	Describe("newLocalStorage", func() {
		Context("with valid path", func() {
			It("should create a localStorage instance with correct path", func() {
				u, err := url.Parse("file://" + tempDir)
				Expect(err).ToNot(HaveOccurred())

				storage := newLocalStorage(*u)
				localStorage := storage.(*localStorage)

				Expect(localStorage.u.Scheme).To(Equal("file"))
				// Check that the path is set correctly (could be resolved to real path on macOS)
				Expect(localStorage.u.Path).To(ContainSubstring("navidrome-local-storage-test"))
				Expect(localStorage.resolvedPath).To(ContainSubstring("navidrome-local-storage-test"))
				Expect(localStorage.extractor).ToNot(BeNil())
			})

			It("should handle URL-decoded paths correctly", func() {
				// Create a directory with spaces to test URL decoding
				spacedDir := filepath.Join(tempDir, "test folder")
				err := os.MkdirAll(spacedDir, 0755)
				Expect(err).ToNot(HaveOccurred())

				// Use proper URL construction instead of manual escaping
				u := &url.URL{
					Scheme: "file",
					Path:   spacedDir,
				}

				storage := newLocalStorage(*u)
				localStorage, ok := storage.(*localStorage)
				Expect(ok).To(BeTrue())

				Expect(localStorage.u.Path).To(Equal(spacedDir))
			})

			It("should resolve symlinks when possible", func() {
				// Create a real directory and a symlink to it
				realDir := filepath.Join(tempDir, "real")
				linkDir := filepath.Join(tempDir, "link")

				err := os.MkdirAll(realDir, 0755)
				Expect(err).ToNot(HaveOccurred())

				err = os.Symlink(realDir, linkDir)
				Expect(err).ToNot(HaveOccurred())

				u, err := url.Parse("file://" + linkDir)
				Expect(err).ToNot(HaveOccurred())

				storage := newLocalStorage(*u)
				localStorage, ok := storage.(*localStorage)
				Expect(ok).To(BeTrue())

				Expect(localStorage.u.Path).To(Equal(linkDir))
				// Check that the resolved path contains the real directory name
				Expect(localStorage.resolvedPath).To(ContainSubstring("real"))
			})

			It("should use u.Path as resolvedPath when symlink resolution fails", func() {
				// Use a non-existent path to trigger symlink resolution failure
				nonExistentPath := filepath.Join(tempDir, "non-existent")

				u, err := url.Parse("file://" + nonExistentPath)
				Expect(err).ToNot(HaveOccurred())

				storage := newLocalStorage(*u)
				localStorage, ok := storage.(*localStorage)
				Expect(ok).To(BeTrue())

				Expect(localStorage.u.Path).To(Equal(nonExistentPath))
				Expect(localStorage.resolvedPath).To(Equal(nonExistentPath))
			})
		})

		Context("with Windows path", func() {
			BeforeEach(func() {
				if runtime.GOOS != "windows" {
					Skip("Windows-specific test")
				}
			})

			It("should handle Windows drive letters correctly", func() {
				u, err := url.Parse("file://C:/music")
				Expect(err).ToNot(HaveOccurred())

				storage := newLocalStorage(*u)
				localStorage, ok := storage.(*localStorage)
				Expect(ok).To(BeTrue())

				Expect(localStorage.u.Path).To(Equal("C:/music"))
			})
		})

		Context("with invalid extractor", func() {
			It("should handle extractor validation correctly", func() {
				// Note: The actual implementation uses log.Fatal which exits the process,
				// so we test the normal path where extractors exist

				u, err := url.Parse("file://" + tempDir)
				Expect(err).ToNot(HaveOccurred())

				storage := newLocalStorage(*u)
				Expect(storage).ToNot(BeNil())
			})
		})
	})

	Describe("localStorage.FS", func() {
		Context("with existing directory", func() {
			It("should return a localFS instance", func() {
				u, err := url.Parse("file://" + tempDir)
				Expect(err).ToNot(HaveOccurred())

				storage := newLocalStorage(*u)
				musicFS, err := storage.FS()
				Expect(err).ToNot(HaveOccurred())
				Expect(musicFS).ToNot(BeNil())

				_, ok := musicFS.(*localFS)
				Expect(ok).To(BeTrue())
			})
		})

		Context("with non-existent directory", func() {
			It("should return an error", func() {
				nonExistentPath := filepath.Join(tempDir, "non-existent")
				u, err := url.Parse("file://" + nonExistentPath)
				Expect(err).ToNot(HaveOccurred())

				storage := newLocalStorage(*u)
				_, err = storage.FS()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring(nonExistentPath))
			})
		})
	})

	Describe("localFS.ReadTags", func() {
		var testFile string

		BeforeEach(func() {
			// Create a test file
			testFile = filepath.Join(tempDir, "test.mp3")
			err := os.WriteFile(testFile, []byte("test data"), 0600)
			Expect(err).ToNot(HaveOccurred())

			// Reset extractor state
			testExtractor.results = make(map[string]metadata.Info)
			testExtractor.err = nil
		})

		Context("when extractor returns complete metadata", func() {
			It("should return the metadata as-is", func() {
				expectedInfo := metadata.Info{
					Tags: map[string][]string{
						"title":  {"Test Song"},
						"artist": {"Test Artist"},
					},
					AudioProperties: metadata.AudioProperties{
						Duration: 180,
						BitRate:  320,
					},
					FileInfo: &testFileInfo{name: "test.mp3"},
				}

				testExtractor.results["test.mp3"] = expectedInfo

				u, err := url.Parse("file://" + tempDir)
				Expect(err).ToNot(HaveOccurred())
				storage := newLocalStorage(*u)
				musicFS, err := storage.FS()
				Expect(err).ToNot(HaveOccurred())

				results, err := musicFS.ReadTags("test.mp3")
				Expect(err).ToNot(HaveOccurred())
				Expect(results).To(HaveKey("test.mp3"))
				Expect(results["test.mp3"]).To(Equal(expectedInfo))
			})
		})

		Context("when extractor returns metadata without FileInfo", func() {
			It("should populate FileInfo from filesystem", func() {
				incompleteInfo := metadata.Info{
					Tags: map[string][]string{
						"title": {"Test Song"},
					},
					FileInfo: nil, // Missing FileInfo
				}

				testExtractor.results["test.mp3"] = incompleteInfo

				u, err := url.Parse("file://" + tempDir)
				Expect(err).ToNot(HaveOccurred())
				storage := newLocalStorage(*u)
				musicFS, err := storage.FS()
				Expect(err).ToNot(HaveOccurred())

				results, err := musicFS.ReadTags("test.mp3")
				Expect(err).ToNot(HaveOccurred())
				Expect(results).To(HaveKey("test.mp3"))

				result := results["test.mp3"]
				Expect(result.FileInfo).ToNot(BeNil())
				Expect(result.FileInfo.Name()).To(Equal("test.mp3"))

				// Should be wrapped in localFileInfo
				_, ok := result.FileInfo.(localFileInfo)
				Expect(ok).To(BeTrue())
			})
		})

		Context("when filesystem stat fails", func() {
			It("should return an error", func() {
				incompleteInfo := metadata.Info{
					Tags:     map[string][]string{"title": {"Test Song"}},
					FileInfo: nil,
				}

				testExtractor.results["non-existent.mp3"] = incompleteInfo

				u, err := url.Parse("file://" + tempDir)
				Expect(err).ToNot(HaveOccurred())
				storage := newLocalStorage(*u)
				musicFS, err := storage.FS()
				Expect(err).ToNot(HaveOccurred())

				_, err = musicFS.ReadTags("non-existent.mp3")
				Expect(err).To(HaveOccurred())
			})
		})

		Context("when extractor fails", func() {
			It("should return the extractor error", func() {
				testExtractor.err = &extractorError{message: "extractor failed"}

				u, err := url.Parse("file://" + tempDir)
				Expect(err).ToNot(HaveOccurred())
				storage := newLocalStorage(*u)
				musicFS, err := storage.FS()
				Expect(err).ToNot(HaveOccurred())

				_, err = musicFS.ReadTags("test.mp3")
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("extractor failed"))
			})
		})

		Context("with multiple files", func() {
			It("should process all files correctly", func() {
				// Create another test file
				testFile2 := filepath.Join(tempDir, "test2.mp3")
				err := os.WriteFile(testFile2, []byte("test data 2"), 0600)
				Expect(err).ToNot(HaveOccurred())

				info1 := metadata.Info{
					Tags:     map[string][]string{"title": {"Song 1"}},
					FileInfo: &testFileInfo{name: "test.mp3"},
				}
				info2 := metadata.Info{
					Tags:     map[string][]string{"title": {"Song 2"}},
					FileInfo: nil, // This one needs FileInfo populated
				}

				testExtractor.results["test.mp3"] = info1
				testExtractor.results["test2.mp3"] = info2

				u, err := url.Parse("file://" + tempDir)
				Expect(err).ToNot(HaveOccurred())
				storage := newLocalStorage(*u)
				musicFS, err := storage.FS()
				Expect(err).ToNot(HaveOccurred())

				results, err := musicFS.ReadTags("test.mp3", "test2.mp3")
				Expect(err).ToNot(HaveOccurred())
				Expect(results).To(HaveLen(2))

				Expect(results["test.mp3"].FileInfo).To(Equal(&testFileInfo{name: "test.mp3"}))
				Expect(results["test2.mp3"].FileInfo).ToNot(BeNil())
				Expect(results["test2.mp3"].FileInfo.Name()).To(Equal("test2.mp3"))
			})
		})
	})

	Describe("localFileInfo", func() {
		var testFile string
		var fileInfo fs.FileInfo

		BeforeEach(func() {
			testFile = filepath.Join(tempDir, "test.mp3")
			err := os.WriteFile(testFile, []byte("test data"), 0600)
			Expect(err).ToNot(HaveOccurred())

			fileInfo, err = os.Stat(testFile)
			Expect(err).ToNot(HaveOccurred())
		})

		Describe("BirthTime", func() {
			It("should return birth time when available", func() {
				lfi := localFileInfo{FileInfo: fileInfo}
				birthTime := lfi.BirthTime()

				// Birth time should be a valid time (not zero value)
				Expect(birthTime).ToNot(BeZero())
				// Should be around the current time (within last few minutes)
				Expect(birthTime).To(BeTemporally("~", time.Now(), 5*time.Minute))
			})
		})

		It("should delegate all other FileInfo methods", func() {
			lfi := localFileInfo{FileInfo: fileInfo}

			Expect(lfi.Name()).To(Equal(fileInfo.Name()))
			Expect(lfi.Size()).To(Equal(fileInfo.Size()))
			Expect(lfi.Mode()).To(Equal(fileInfo.Mode()))
			Expect(lfi.ModTime()).To(Equal(fileInfo.ModTime()))
			Expect(lfi.IsDir()).To(Equal(fileInfo.IsDir()))
			Expect(lfi.Sys()).To(Equal(fileInfo.Sys()))
		})
	})

	Describe("Storage registration", func() {
		It("should register localStorage for file scheme", func() {
			// This tests the init() function indirectly
			storage, err := storage.For("file://" + tempDir)
			Expect(err).ToNot(HaveOccurred())
			Expect(storage).To(BeAssignableToTypeOf(&localStorage{}))
		})
	})
})

// Test extractor for testing
type mockTestExtractor struct {
	results map[string]metadata.Info
	err     error
}

func (m *mockTestExtractor) Parse(files ...string) (map[string]metadata.Info, error) {
	if m.err != nil {
		return nil, m.err
	}

	result := make(map[string]metadata.Info)
	for _, file := range files {
		if info, exists := m.results[file]; exists {
			result[file] = info
		}
	}
	return result, nil
}

func (m *mockTestExtractor) Version() string {
	return "test-1.0"
}

type extractorError struct {
	message string
}

func (e *extractorError) Error() string {
	return e.message
}

// Test FileInfo that implements metadata.FileInfo
type testFileInfo struct {
	name      string
	size      int64
	mode      fs.FileMode
	modTime   time.Time
	isDir     bool
	birthTime time.Time
}

func (t *testFileInfo) Name() string       { return t.name }
func (t *testFileInfo) Size() int64        { return t.size }
func (t *testFileInfo) Mode() fs.FileMode  { return t.mode }
func (t *testFileInfo) ModTime() time.Time { return t.modTime }
func (t *testFileInfo) IsDir() bool        { return t.isDir }
func (t *testFileInfo) Sys() any           { return nil }
func (t *testFileInfo) BirthTime() time.Time {
	if t.birthTime.IsZero() {
		return time.Now()
	}
	return t.birthTime
}
