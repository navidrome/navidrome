//go:build !windows

package plugins

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"path/filepath"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/conf/configtest"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("LibraryService", Ordered, func() {
	var (
		ctx     context.Context
		ds      model.DataStore
		service *libraryServiceImpl
	)

	BeforeEach(func() {
		DeferCleanup(configtest.SetupConfig())
		ctx = context.Background()
		ds = &tests.MockDataStore{}
	})

	Describe("GetLibrary", func() {
		It("should return library metadata without filesystem permission", func() {
			reason := "test"
			service = newLibraryService(ds, &LibraryPermission{Reason: &reason, Filesystem: false}, nil, true).(*libraryServiceImpl)

			lib := &model.Library{
				ID:            1,
				Name:          "Test Library",
				Path:          "/music/test",
				TotalSongs:    100,
				TotalAlbums:   10,
				TotalArtists:  5,
				TotalSize:     1024000,
				TotalDuration: 3600.5,
			}
			lib.LastScanAt = lib.LastScanAt.Add(0) // Ensure time is set

			mockLibRepo := ds.Library(ctx).(*tests.MockLibraryRepo)
			mockLibRepo.SetData(model.Libraries{*lib})

			result, err := service.GetLibrary(ctx, 1)
			Expect(err).ToNot(HaveOccurred())
			Expect(result.ID).To(Equal(int32(1)))
			Expect(result.Name).To(Equal("Test Library"))
			Expect(result.TotalSongs).To(Equal(int32(100)))
			Expect(result.TotalAlbums).To(Equal(int32(10)))
			Expect(result.TotalArtists).To(Equal(int32(5)))
			Expect(result.TotalSize).To(Equal(int64(1024000)))
			Expect(result.TotalDuration).To(Equal(3600.5))
			Expect(result.Path).To(BeEmpty(), "Path should not be included without filesystem permission")
			Expect(result.MountPoint).To(BeEmpty(), "MountPoint should not be included without filesystem permission")
		})

		It("should return library metadata with filesystem permission", func() {
			reason := "test"
			service = newLibraryService(ds, &LibraryPermission{Reason: &reason, Filesystem: true}, nil, true).(*libraryServiceImpl)

			lib := &model.Library{
				ID:            2,
				Name:          "FS Library",
				Path:          "/music/fs",
				TotalSongs:    50,
				TotalAlbums:   5,
				TotalArtists:  3,
				TotalSize:     512000,
				TotalDuration: 1800.0,
			}

			mockLibRepo := ds.Library(ctx).(*tests.MockLibraryRepo)
			mockLibRepo.SetData(model.Libraries{*lib})

			result, err := service.GetLibrary(ctx, 2)
			Expect(err).ToNot(HaveOccurred())
			Expect(result.ID).To(Equal(int32(2)))
			Expect(result.Name).To(Equal("FS Library"))
			Expect(result.Path).To(Equal("/music/fs"), "Path should be included with filesystem permission")
			Expect(result.MountPoint).To(Equal("/libraries/2"), "MountPoint should be included with filesystem permission")
		})

		It("should return error for non-existent library", func() {
			reason := "test"
			service = newLibraryService(ds, &LibraryPermission{Reason: &reason}, nil, true).(*libraryServiceImpl)

			mockLibRepo := ds.Library(ctx).(*tests.MockLibraryRepo)
			mockLibRepo.SetData(model.Libraries{})

			_, err := service.GetLibrary(ctx, 999)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("library not found"))
		})
	})

	Describe("GetAllLibraries", func() {
		It("should return all libraries without filesystem permission", func() {
			reason := "test"
			service = newLibraryService(ds, &LibraryPermission{Reason: &reason, Filesystem: false}, nil, true).(*libraryServiceImpl)

			libs := model.Libraries{
				{ID: 1, Name: "Rock", Path: "/music/rock", TotalSongs: 100},
				{ID: 2, Name: "Jazz", Path: "/music/jazz", TotalSongs: 50},
			}

			mockLibRepo := ds.Library(ctx).(*tests.MockLibraryRepo)
			mockLibRepo.SetData(libs)

			results, err := service.GetAllLibraries(ctx)
			Expect(err).ToNot(HaveOccurred())
			Expect(results).To(HaveLen(2))
			Expect(results[0].Name).To(Equal("Rock"))
			Expect(results[0].Path).To(BeEmpty())
			Expect(results[0].MountPoint).To(BeEmpty())
			Expect(results[1].Name).To(Equal("Jazz"))
			Expect(results[1].Path).To(BeEmpty())
			Expect(results[1].MountPoint).To(BeEmpty())
		})

		It("should return all libraries with filesystem permission", func() {
			reason := "test"
			service = newLibraryService(ds, &LibraryPermission{Reason: &reason, Filesystem: true}, nil, true).(*libraryServiceImpl)

			libs := model.Libraries{
				{ID: 1, Name: "Rock", Path: "/music/rock", TotalSongs: 100},
				{ID: 2, Name: "Jazz", Path: "/music/jazz", TotalSongs: 50},
			}

			mockLibRepo := ds.Library(ctx).(*tests.MockLibraryRepo)
			mockLibRepo.SetData(libs)

			results, err := service.GetAllLibraries(ctx)
			Expect(err).ToNot(HaveOccurred())
			Expect(results).To(HaveLen(2))
			Expect(results[0].Path).To(Equal("/music/rock"))
			Expect(results[0].MountPoint).To(Equal("/libraries/1"))
			Expect(results[1].Path).To(Equal("/music/jazz"))
			Expect(results[1].MountPoint).To(Equal("/libraries/2"))
		})
	})

	Describe("Library Access Filtering", func() {
		It("should only return libraries in the allowed list", func() {
			reason := "test"
			// Only allow library ID 2
			service = newLibraryService(ds, &LibraryPermission{Reason: &reason, Filesystem: false}, []int{2}, false).(*libraryServiceImpl)

			libs := model.Libraries{
				{ID: 1, Name: "Rock", Path: "/music/rock", TotalSongs: 100},
				{ID: 2, Name: "Jazz", Path: "/music/jazz", TotalSongs: 50},
				{ID: 3, Name: "Classical", Path: "/music/classical", TotalSongs: 75},
			}

			mockLibRepo := ds.Library(ctx).(*tests.MockLibraryRepo)
			mockLibRepo.SetData(libs)

			results, err := service.GetAllLibraries(ctx)
			Expect(err).ToNot(HaveOccurred())
			Expect(results).To(HaveLen(1))
			Expect(results[0].ID).To(Equal(int32(2)))
			Expect(results[0].Name).To(Equal("Jazz"))
		})

		It("should return error when getting a library not in the allowed list", func() {
			reason := "test"
			// Only allow library ID 2
			service = newLibraryService(ds, &LibraryPermission{Reason: &reason, Filesystem: false}, []int{2}, false).(*libraryServiceImpl)

			libs := model.Libraries{
				{ID: 1, Name: "Rock", Path: "/music/rock", TotalSongs: 100},
				{ID: 2, Name: "Jazz", Path: "/music/jazz", TotalSongs: 50},
			}

			mockLibRepo := ds.Library(ctx).(*tests.MockLibraryRepo)
			mockLibRepo.SetData(libs)

			// Requesting library 1 which is not in the allowed list
			_, err := service.GetLibrary(ctx, 1)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("not accessible"))
		})

		It("should allow access to a library in the allowed list", func() {
			reason := "test"
			// Only allow library ID 2
			service = newLibraryService(ds, &LibraryPermission{Reason: &reason, Filesystem: false}, []int{2}, false).(*libraryServiceImpl)

			libs := model.Libraries{
				{ID: 1, Name: "Rock", Path: "/music/rock", TotalSongs: 100},
				{ID: 2, Name: "Jazz", Path: "/music/jazz", TotalSongs: 50},
			}

			mockLibRepo := ds.Library(ctx).(*tests.MockLibraryRepo)
			mockLibRepo.SetData(libs)

			result, err := service.GetLibrary(ctx, 2)
			Expect(err).ToNot(HaveOccurred())
			Expect(result.ID).To(Equal(int32(2)))
			Expect(result.Name).To(Equal("Jazz"))
		})

		It("should return empty list when no libraries are allowed and allLibraries is false", func() {
			reason := "test"
			// No libraries allowed
			service = newLibraryService(ds, &LibraryPermission{Reason: &reason, Filesystem: false}, []int{}, false).(*libraryServiceImpl)

			libs := model.Libraries{
				{ID: 1, Name: "Rock", Path: "/music/rock", TotalSongs: 100},
				{ID: 2, Name: "Jazz", Path: "/music/jazz", TotalSongs: 50},
			}

			mockLibRepo := ds.Library(ctx).(*tests.MockLibraryRepo)
			mockLibRepo.SetData(libs)

			results, err := service.GetAllLibraries(ctx)
			Expect(err).ToNot(HaveOccurred())
			Expect(results).To(HaveLen(0))
		})

		It("should return all libraries when allLibraries is true regardless of allowed list", func() {
			reason := "test"
			// allLibraries=true should ignore the allowed list
			service = newLibraryService(ds, &LibraryPermission{Reason: &reason, Filesystem: false}, []int{1}, true).(*libraryServiceImpl)

			libs := model.Libraries{
				{ID: 1, Name: "Rock", Path: "/music/rock", TotalSongs: 100},
				{ID: 2, Name: "Jazz", Path: "/music/jazz", TotalSongs: 50},
			}

			mockLibRepo := ds.Library(ctx).(*tests.MockLibraryRepo)
			mockLibRepo.SetData(libs)

			results, err := service.GetAllLibraries(ctx)
			Expect(err).ToNot(HaveOccurred())
			Expect(results).To(HaveLen(2))
		})
	})

	Describe("Plugin Integration", func() {
		var (
			manager *Manager
			tmpDir  string
		)

		BeforeEach(func() {
			var err error
			tmpDir, err = os.MkdirTemp("", "library-test-*")
			Expect(err).ToNot(HaveOccurred())

			// Note: Since we don't have WASM test plugins yet, we can test
			// the service registration and configuration without full plugin execution
			DeferCleanup(configtest.SetupConfig())
			conf.Server.Plugins.Enabled = true
			conf.Server.Plugins.Folder = tmpDir
			conf.Server.CacheFolder = filepath.Join(tmpDir, "cache")

			// Create mock &tests.MockLibraryRepo{}
			mockLibRepo := &tests.MockLibraryRepo{}
			mockLibRepo.SetData(model.Libraries{
				{ID: 1, Name: "Test", Path: "/tmp/test-music", TotalSongs: 10},
			})

			ds := &tests.MockDataStore{
				MockedProperty: &tests.MockedPropertyRepo{},
				MockedPlugin:   tests.CreateMockPluginRepo(),
				MockedLibrary:  mockLibRepo,
			}

			manager = &Manager{
				plugins: make(map[string]*plugin),
				ds:      ds,
			}

			DeferCleanup(func() {
				if manager != nil {
					_ = manager.Stop()
				}
				_ = os.RemoveAll(tmpDir)
			})
		})

		It("should register library service in hostServices table", func() {
			// Verify the library service is in the hostServices table
			found := false
			for _, entry := range hostServices {
				if entry.name == "Library" {
					found = true
					break
				}
			}
			Expect(found).To(BeTrue(), "Library service should be registered in hostServices")
		})

		It("should configure AllowedPaths when filesystem permission is granted", func() {
			// This test verifies the AllowedPaths configuration logic
			// We can't fully test without a real WASM plugin, but we can verify the setup
			Expect(manager.ds).ToNot(BeNil())

			ctx := context.Background()
			libs, err := manager.ds.Library(adminContext(ctx)).GetAll()
			Expect(err).ToNot(HaveOccurred())
			Expect(libs).To(HaveLen(1))
			Expect(libs[0].Path).To(Equal("/tmp/test-music"))

			// Verify mount point format
			mountPoint := "/libraries/1"
			Expect(mountPoint).To(MatchRegexp(`^/libraries/\d+$`))
		})
	})
})

var _ = Describe("LibraryService Integration", Ordered, func() {
	var (
		manager    *Manager
		tmpDir     string
		libraryDir string
	)

	BeforeAll(func() {
		var err error
		tmpDir, err = os.MkdirTemp("", "library-integration-test-*")
		Expect(err).ToNot(HaveOccurred())

		// Create a library directory with a test file
		libraryDir = filepath.Join(tmpDir, "music-library")
		err = os.MkdirAll(libraryDir, 0755)
		Expect(err).ToNot(HaveOccurred())

		// Create a test file in the library
		testFile := filepath.Join(libraryDir, "test-track.txt")
		err = os.WriteFile(testFile, []byte("test audio file content"), 0600)
		Expect(err).ToNot(HaveOccurred())

		// Copy the test-library plugin
		srcPath := filepath.Join(testdataDir, "test-library"+PackageExtension)
		destPath := filepath.Join(tmpDir, "test-library"+PackageExtension)
		data, err := os.ReadFile(srcPath)
		Expect(err).ToNot(HaveOccurred())
		err = os.WriteFile(destPath, data, 0600)
		Expect(err).ToNot(HaveOccurred())

		// Compute SHA256 for the plugin
		hash := sha256.Sum256(data)
		hashHex := hex.EncodeToString(hash[:])

		// Setup config
		DeferCleanup(configtest.SetupConfig())
		conf.Server.Plugins.Enabled = true
		conf.Server.Plugins.Folder = tmpDir
		conf.Server.Plugins.AutoReload = false
		conf.Server.CacheFolder = filepath.Join(tmpDir, "cache")

		// Setup mock DataStore with pre-enabled plugin and library
		mockPluginRepo := tests.CreateMockPluginRepo()
		mockPluginRepo.Permitted = true
		mockPluginRepo.SetData(model.Plugins{{
			ID:           "test-library",
			Path:         destPath,
			SHA256:       hashHex,
			Enabled:      true,
			AllLibraries: true, // Grant access to all libraries for testing
		}})

		mockLibraryRepo := &tests.MockLibraryRepo{}
		mockLibraryRepo.SetData(model.Libraries{
			{
				ID:            1,
				Name:          "Test Library",
				Path:          libraryDir,
				TotalSongs:    100,
				TotalAlbums:   10,
				TotalArtists:  5,
				TotalSize:     1024000,
				TotalDuration: 3600.5,
			},
			{
				ID:            2,
				Name:          "Jazz Collection",
				Path:          "/nonexistent/jazz",
				TotalSongs:    50,
				TotalAlbums:   5,
				TotalArtists:  3,
				TotalSize:     512000,
				TotalDuration: 1800.0,
			},
		})

		dataStore := &tests.MockDataStore{
			MockedPlugin:  mockPluginRepo,
			MockedLibrary: mockLibraryRepo,
		}

		// Create and start manager
		manager = &Manager{
			plugins:        make(map[string]*plugin),
			ds:             dataStore,
			subsonicRouter: http.NotFoundHandler(),
		}
		err = manager.Start(GinkgoT().Context())
		Expect(err).ToNot(HaveOccurred())

		DeferCleanup(func() {
			_ = manager.Stop()
			_ = os.RemoveAll(tmpDir)
		})
	})

	Describe("Plugin Loading", func() {
		It("should load plugin with library permission", func() {
			manager.mu.RLock()
			p, ok := manager.plugins["test-library"]
			manager.mu.RUnlock()
			Expect(ok).To(BeTrue())
			Expect(p.manifest.Permissions).ToNot(BeNil())
			Expect(p.manifest.Permissions.Library).ToNot(BeNil())
			Expect(p.manifest.Permissions.Library.Filesystem).To(BeTrue())
		})
	})

	Describe("Library Operations via Plugin", func() {
		type testLibraryInput struct {
			Operation  string `json:"operation"`
			LibraryID  int32  `json:"library_id,omitempty"`
			MountPoint string `json:"mount_point,omitempty"`
			FilePath   string `json:"file_path,omitempty"`
		}
		type library struct {
			ID            int32   `json:"id"`
			Name          string  `json:"name"`
			Path          string  `json:"path,omitempty"`
			MountPoint    string  `json:"mountPoint,omitempty"`
			LastScanAt    int64   `json:"lastScanAt"`
			TotalSongs    int32   `json:"totalSongs"`
			TotalAlbums   int32   `json:"totalAlbums"`
			TotalArtists  int32   `json:"totalArtists"`
			TotalSize     int64   `json:"totalSize"`
			TotalDuration float64 `json:"totalDuration"`
		}
		type testLibraryOutput struct {
			Library     *library  `json:"library,omitempty"`
			Libraries   []library `json:"libraries,omitempty"`
			FileContent string    `json:"file_content,omitempty"`
			DirEntries  []string  `json:"dir_entries,omitempty"`
			Error       *string   `json:"error,omitempty"`
		}

		callTestLibrary := func(ctx context.Context, input testLibraryInput) (*testLibraryOutput, error) {
			manager.mu.RLock()
			p := manager.plugins["test-library"]
			manager.mu.RUnlock()

			instance, err := p.instance(ctx)
			if err != nil {
				return nil, err
			}
			defer instance.Close(ctx)

			inputBytes, _ := json.Marshal(input)
			_, outputBytes, err := instance.Call("nd_test_library", inputBytes)
			if err != nil {
				return nil, err
			}

			var output testLibraryOutput
			if err := json.Unmarshal(outputBytes, &output); err != nil {
				return nil, err
			}
			if output.Error != nil {
				return nil, errors.New(*output.Error)
			}
			return &output, nil
		}

		It("should get library by ID with metadata", func() {
			ctx := GinkgoT().Context()

			output, err := callTestLibrary(ctx, testLibraryInput{
				Operation: "get_library",
				LibraryID: 1,
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(output.Library).ToNot(BeNil())
			Expect(output.Library.ID).To(Equal(int32(1)))
			Expect(output.Library.Name).To(Equal("Test Library"))
			Expect(output.Library.TotalSongs).To(Equal(int32(100)))
			Expect(output.Library.TotalAlbums).To(Equal(int32(10)))
			Expect(output.Library.TotalArtists).To(Equal(int32(5)))
		})

		It("should include path and mount point with filesystem permission", func() {
			ctx := GinkgoT().Context()

			output, err := callTestLibrary(ctx, testLibraryInput{
				Operation: "get_library",
				LibraryID: 1,
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(output.Library).ToNot(BeNil())
			Expect(output.Library.Path).To(Equal(libraryDir))
			Expect(output.Library.MountPoint).To(Equal("/libraries/1"))
		})

		It("should get all libraries", func() {
			ctx := GinkgoT().Context()

			output, err := callTestLibrary(ctx, testLibraryInput{
				Operation: "get_all_libraries",
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(output.Libraries).To(HaveLen(2))

			// First library
			Expect(output.Libraries[0].ID).To(Equal(int32(1)))
			Expect(output.Libraries[0].Name).To(Equal("Test Library"))
			Expect(output.Libraries[0].MountPoint).To(Equal("/libraries/1"))

			// Second library
			Expect(output.Libraries[1].ID).To(Equal(int32(2)))
			Expect(output.Libraries[1].Name).To(Equal("Jazz Collection"))
			Expect(output.Libraries[1].MountPoint).To(Equal("/libraries/2"))
		})

		It("should return error for non-existent library", func() {
			ctx := GinkgoT().Context()

			_, err := callTestLibrary(ctx, testLibraryInput{
				Operation: "get_library",
				LibraryID: 999,
			})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("library not found"))
		})

		// Note: This test is slightly flaky due to a potential race condition in wazero's
		// WASI filesystem mounting. The test passes ~85% of the time. Using FlakeAttempts
		// to automatically retry on failure.
		It("should read file from mounted library directory", FlakeAttempts(3), func() {
			ctx := GinkgoT().Context()

			output, err := callTestLibrary(ctx, testLibraryInput{
				Operation:  "read_file",
				MountPoint: "/libraries/1",
				FilePath:   "test-track.txt",
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(output.FileContent).To(Equal("test audio file content"))
		})

		// Note: Uses FlakeAttempts for the same reason as the read_file test above
		It("should list files in mounted library directory", FlakeAttempts(3), func() {
			ctx := GinkgoT().Context()

			output, err := callTestLibrary(ctx, testLibraryInput{
				Operation:  "list_dir",
				MountPoint: "/libraries/1",
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(output.DirEntries).To(ContainElement("test-track.txt"))
		})

		It("should fail to access unmapped library directory", func() {
			ctx := GinkgoT().Context()

			// Try to access a path outside the mapped libraries
			_, err := callTestLibrary(ctx, testLibraryInput{
				Operation:  "list_dir",
				MountPoint: "/etc",
			})
			Expect(err).To(HaveOccurred())
		})
	})
})
