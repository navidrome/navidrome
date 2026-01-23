package scanner

import (
	"context"
	"io/fs"
	"path/filepath"
	"testing/fstest"
	"time"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/conf/configtest"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Watcher", func() {
	var ctx context.Context
	var cancel context.CancelFunc
	var mockScanner *tests.MockScanner
	var mockDS *tests.MockDataStore
	var w *watcher
	var lib *model.Library

	BeforeEach(func() {
		DeferCleanup(configtest.SetupConfig())
		conf.Server.Scanner.WatcherWait = 50 * time.Millisecond // Short wait for tests

		ctx, cancel = context.WithCancel(context.Background())
		DeferCleanup(cancel)

		lib = &model.Library{
			ID:   1,
			Name: "Test Library",
			Path: "/test/library",
		}

		// Set up mocks
		mockScanner = tests.NewMockScanner()
		mockDS = &tests.MockDataStore{}
		mockLibRepo := &tests.MockLibraryRepo{}
		mockLibRepo.SetData(model.Libraries{*lib})
		mockDS.MockedLibrary = mockLibRepo

		// Create a new watcher instance (not singleton) for testing
		w = &watcher{
			ds:              mockDS,
			scanner:         mockScanner,
			triggerWait:     conf.Server.Scanner.WatcherWait,
			watcherNotify:   make(chan scanNotification, 10),
			libraryWatchers: make(map[int]*libraryWatcherInstance),
			mainCtx:         ctx,
		}
	})

	Describe("Target Collection and Deduplication", func() {
		BeforeEach(func() {
			// Start watcher in background
			go func() {
				_ = w.Run(ctx)
			}()

			// Give watcher time to initialize
			time.Sleep(10 * time.Millisecond)
		})

		It("creates separate targets for different folders", func() {
			// Send notifications for different folders
			w.watcherNotify <- scanNotification{Library: lib, FolderPath: "artist1"}
			time.Sleep(10 * time.Millisecond)
			w.watcherNotify <- scanNotification{Library: lib, FolderPath: "artist2"}

			// Wait for watcher to process and trigger scan
			Eventually(func() int {
				return mockScanner.GetScanFoldersCallCount()
			}, 200*time.Millisecond, 10*time.Millisecond).Should(Equal(1))

			// Verify two targets
			calls := mockScanner.GetScanFoldersCalls()
			Expect(calls).To(HaveLen(1))
			Expect(calls[0].Targets).To(HaveLen(2))

			// Extract folder paths
			folderPaths := make(map[string]bool)
			for _, target := range calls[0].Targets {
				Expect(target.LibraryID).To(Equal(1))
				folderPaths[target.FolderPath] = true
			}
			Expect(folderPaths).To(HaveKey("artist1"))
			Expect(folderPaths).To(HaveKey("artist2"))
		})

		It("handles different folder paths correctly", func() {
			// Send notification for nested folder
			w.watcherNotify <- scanNotification{Library: lib, FolderPath: "artist1/album1"}

			// Wait for watcher to process and trigger scan
			Eventually(func() int {
				return mockScanner.GetScanFoldersCallCount()
			}, 200*time.Millisecond, 10*time.Millisecond).Should(Equal(1))

			// Verify the target
			calls := mockScanner.GetScanFoldersCalls()
			Expect(calls).To(HaveLen(1))
			Expect(calls[0].Targets).To(HaveLen(1))
			Expect(calls[0].Targets[0].FolderPath).To(Equal("artist1/album1"))
		})

		It("deduplicates folder and file within same folder", func() {
			// Send notification for a folder
			w.watcherNotify <- scanNotification{Library: lib, FolderPath: "artist1/album1"}
			time.Sleep(10 * time.Millisecond)
			// Send notification for same folder (as if file change was detected there)
			// In practice, watchLibrary() would walk up from file path to folder
			w.watcherNotify <- scanNotification{Library: lib, FolderPath: "artist1/album1"}
			time.Sleep(10 * time.Millisecond)
			// Send another for same folder
			w.watcherNotify <- scanNotification{Library: lib, FolderPath: "artist1/album1"}

			// Wait for watcher to process and trigger scan
			Eventually(func() int {
				return mockScanner.GetScanFoldersCallCount()
			}, 200*time.Millisecond, 10*time.Millisecond).Should(Equal(1))

			// Verify only one target despite multiple file/folder changes
			calls := mockScanner.GetScanFoldersCalls()
			Expect(calls).To(HaveLen(1))
			Expect(calls[0].Targets).To(HaveLen(1))
			Expect(calls[0].Targets[0].FolderPath).To(Equal("artist1/album1"))
		})
	})

	Describe("Timer Behavior", func() {
		BeforeEach(func() {
			// Start watcher in background
			go func() {
				_ = w.Run(ctx)
			}()

			// Give watcher time to initialize
			time.Sleep(10 * time.Millisecond)
		})

		It("resets timer on each change (debouncing)", func() {
			// Send first notification
			w.watcherNotify <- scanNotification{Library: lib, FolderPath: "artist1"}

			// Wait a bit less than half the watcher wait time to ensure timer doesn't fire
			time.Sleep(20 * time.Millisecond)

			// No scan should have been triggered yet
			Expect(mockScanner.GetScanFoldersCallCount()).To(Equal(0))

			// Send another notification (resets timer)
			w.watcherNotify <- scanNotification{Library: lib, FolderPath: "artist1"}

			// Wait a bit less than half the watcher wait time again
			time.Sleep(20 * time.Millisecond)

			// Still no scan
			Expect(mockScanner.GetScanFoldersCallCount()).To(Equal(0))

			// Wait for full timer to expire after last notification (plus margin)
			time.Sleep(60 * time.Millisecond)

			// Now scan should have been triggered
			Eventually(func() int {
				return mockScanner.GetScanFoldersCallCount()
			}, 100*time.Millisecond, 10*time.Millisecond).Should(Equal(1))
		})

		It("triggers scan after quiet period", func() {
			// Send notification
			w.watcherNotify <- scanNotification{Library: lib, FolderPath: "artist1"}

			// No scan immediately
			Expect(mockScanner.GetScanFoldersCallCount()).To(Equal(0))

			// Wait for quiet period
			Eventually(func() int {
				return mockScanner.GetScanFoldersCallCount()
			}, 200*time.Millisecond, 10*time.Millisecond).Should(Equal(1))
		})
	})

	Describe("Empty and Root Paths", func() {
		BeforeEach(func() {
			// Start watcher in background
			go func() {
				_ = w.Run(ctx)
			}()

			// Give watcher time to initialize
			time.Sleep(10 * time.Millisecond)
		})

		It("handles empty folder path (library root)", func() {
			// Send notification with empty folder path
			w.watcherNotify <- scanNotification{Library: lib, FolderPath: ""}

			// Wait for scan
			Eventually(func() int {
				return mockScanner.GetScanFoldersCallCount()
			}, 200*time.Millisecond, 10*time.Millisecond).Should(Equal(1))

			// Should scan the library root
			calls := mockScanner.GetScanFoldersCalls()
			Expect(calls).To(HaveLen(1))
			Expect(calls[0].Targets).To(HaveLen(1))
			Expect(calls[0].Targets[0].FolderPath).To(Equal(""))
		})

		It("deduplicates empty and dot paths", func() {
			// Send notifications with empty and dot paths
			w.watcherNotify <- scanNotification{Library: lib, FolderPath: ""}
			time.Sleep(10 * time.Millisecond)
			w.watcherNotify <- scanNotification{Library: lib, FolderPath: ""}

			// Wait for scan
			Eventually(func() int {
				return mockScanner.GetScanFoldersCallCount()
			}, 200*time.Millisecond, 10*time.Millisecond).Should(Equal(1))

			// Should have only one target
			calls := mockScanner.GetScanFoldersCalls()
			Expect(calls).To(HaveLen(1))
			Expect(calls[0].Targets).To(HaveLen(1))
		})
	})

	Describe("Multiple Libraries", func() {
		var lib2 *model.Library

		BeforeEach(func() {
			// Create second library
			lib2 = &model.Library{
				ID:   2,
				Name: "Test Library 2",
				Path: "/test/library2",
			}

			mockLibRepo := mockDS.MockedLibrary.(*tests.MockLibraryRepo)
			mockLibRepo.SetData(model.Libraries{*lib, *lib2})

			// Start watcher in background
			go func() {
				_ = w.Run(ctx)
			}()

			// Give watcher time to initialize
			time.Sleep(10 * time.Millisecond)
		})

		It("creates separate targets for different libraries", func() {
			// Send notifications for both libraries
			w.watcherNotify <- scanNotification{Library: lib, FolderPath: "artist1"}
			time.Sleep(10 * time.Millisecond)
			w.watcherNotify <- scanNotification{Library: lib2, FolderPath: "artist2"}

			// Wait for scan
			Eventually(func() int {
				return mockScanner.GetScanFoldersCallCount()
			}, 200*time.Millisecond, 10*time.Millisecond).Should(Equal(1))

			// Verify two targets for different libraries
			calls := mockScanner.GetScanFoldersCalls()
			Expect(calls).To(HaveLen(1))
			Expect(calls[0].Targets).To(HaveLen(2))

			// Verify library IDs are different
			libraryIDs := make(map[int]bool)
			for _, target := range calls[0].Targets {
				libraryIDs[target.LibraryID] = true
			}
			Expect(libraryIDs).To(HaveKey(1))
			Expect(libraryIDs).To(HaveKey(2))
		})
	})

	Describe(".ndignore handling", func() {
		var ctx context.Context
		var cancel context.CancelFunc
		var w *watcher
		var mockFS *mockMusicFS
		var lib *model.Library
		var eventChan chan string
		var absLibPath string

		BeforeEach(func() {
			ctx, cancel = context.WithCancel(GinkgoT().Context())
			DeferCleanup(cancel)

			// Set up library
			var err error
			absLibPath, err = filepath.Abs(".")
			Expect(err).NotTo(HaveOccurred())

			lib = &model.Library{
				ID:   1,
				Name: "Test Library",
				Path: absLibPath,
			}

			// Create watcher with notification channel
			w = &watcher{
				watcherNotify: make(chan scanNotification, 10),
			}

			eventChan = make(chan string, 10)
		})

		// Helper to send an event - converts relative path to absolute
		sendEvent := func(relativePath string) {
			path := filepath.Join(absLibPath, relativePath)
			eventChan <- path
		}

		// Helper to start the real event processing loop
		startEventProcessing := func() {
			go func() {
				defer GinkgoRecover()
				// Call the actual processLibraryEvents method - testing the real implementation!
				_ = w.processLibraryEvents(ctx, lib, mockFS, eventChan, absLibPath)
			}()
		}

		Context("when a folder matching .ndignore is deleted", func() {
			BeforeEach(func() {
				// Create filesystem with .ndignore containing _TEMP pattern
				// The deleted folder (_TEMP) will NOT exist in the filesystem
				mockFS = &mockMusicFS{
					FS: fstest.MapFS{
						"rock":                       &fstest.MapFile{Mode: fs.ModeDir},
						"rock/.ndignore":             &fstest.MapFile{Data: []byte("_TEMP\n")},
						"rock/valid_album":           &fstest.MapFile{Mode: fs.ModeDir},
						"rock/valid_album/track.mp3": &fstest.MapFile{Data: []byte("audio")},
					},
				}
			})

			It("should NOT send scan notification when deleted folder matches .ndignore", func() {
				startEventProcessing()

				// Simulate deletion event for rock/_TEMP
				sendEvent("rock/_TEMP")

				// Wait a bit to ensure event is processed
				time.Sleep(50 * time.Millisecond)

				// No notification should have been sent
				Consistently(eventChan, 100*time.Millisecond).Should(BeEmpty())
			})

			It("should send scan notification for valid folder deletion", func() {
				startEventProcessing()

				// Simulate deletion event for rock/other_folder (not in .ndignore and doesn't exist)
				// Since it doesn't exist in mockFS, resolveFolderPath will walk up to "rock"
				sendEvent("rock/other_folder")

				// Should receive notification for parent folder
				Eventually(w.watcherNotify, 200*time.Millisecond).Should(Receive(Equal(scanNotification{
					Library:    lib,
					FolderPath: "rock",
				})))
			})
		})

		Context("with nested folder patterns", func() {
			BeforeEach(func() {
				mockFS = &mockMusicFS{
					FS: fstest.MapFS{
						"music":             &fstest.MapFile{Mode: fs.ModeDir},
						"music/.ndignore":   &fstest.MapFile{Data: []byte("**/temp\n**/cache\n")},
						"music/rock":        &fstest.MapFile{Mode: fs.ModeDir},
						"music/rock/artist": &fstest.MapFile{Mode: fs.ModeDir},
					},
				}
			})

			It("should NOT send notification when nested ignored folder is deleted", func() {
				startEventProcessing()

				// Simulate deletion of music/rock/artist/temp (matches **/temp)
				sendEvent("music/rock/artist/temp")

				// Wait to ensure event is processed
				time.Sleep(50 * time.Millisecond)

				// No notification should be sent
				Expect(w.watcherNotify).To(BeEmpty(), "Expected no scan notification for nested ignored folder")
			})

			It("should send notification for non-ignored nested folder", func() {
				startEventProcessing()

				// Simulate change in music/rock/artist (doesn't match any pattern)
				sendEvent("music/rock/artist")

				// Should receive notification
				Eventually(w.watcherNotify, 200*time.Millisecond).Should(Receive(Equal(scanNotification{
					Library:    lib,
					FolderPath: "music/rock/artist",
				})))
			})
		})

		Context("with file events in ignored folders", func() {
			BeforeEach(func() {
				mockFS = &mockMusicFS{
					FS: fstest.MapFS{
						"rock":           &fstest.MapFile{Mode: fs.ModeDir},
						"rock/.ndignore": &fstest.MapFile{Data: []byte("_TEMP\n")},
					},
				}
			})

			It("should NOT send notification for file changes in ignored folders", func() {
				startEventProcessing()

				// Simulate file change in rock/_TEMP/file.mp3
				sendEvent("rock/_TEMP/file.mp3")

				// Wait to ensure event is processed
				time.Sleep(50 * time.Millisecond)

				// No notification should be sent
				Expect(w.watcherNotify).To(BeEmpty(), "Expected no scan notification for file in ignored folder")
			})
		})
	})
})

var _ = Describe("resolveFolderPath", func() {
	var mockFS fs.FS

	BeforeEach(func() {
		// Create a mock filesystem with some directories and files
		mockFS = fstest.MapFS{
			"artist1":                   &fstest.MapFile{Mode: fs.ModeDir},
			"artist1/album1":            &fstest.MapFile{Mode: fs.ModeDir},
			"artist1/album1/track1.mp3": &fstest.MapFile{Data: []byte("audio")},
			"artist1/album1/track2.mp3": &fstest.MapFile{Data: []byte("audio")},
			"artist1/album2":            &fstest.MapFile{Mode: fs.ModeDir},
			"artist1/album2/song.flac":  &fstest.MapFile{Data: []byte("audio")},
			"artist2":                   &fstest.MapFile{Mode: fs.ModeDir},
			"artist2/cover.jpg":         &fstest.MapFile{Data: []byte("image")},
		}
	})

	It("returns directory path when given a directory", func() {
		result := resolveFolderPath(mockFS, "artist1/album1")
		Expect(result).To(Equal("artist1/album1"))
	})

	It("walks up to parent directory when given a file path", func() {
		result := resolveFolderPath(mockFS, "artist1/album1/track1.mp3")
		Expect(result).To(Equal("artist1/album1"))
	})

	It("walks up multiple levels if needed", func() {
		result := resolveFolderPath(mockFS, "artist1/album1/nonexistent/file.mp3")
		Expect(result).To(Equal("artist1/album1"))
	})

	It("returns empty string for non-existent paths at root", func() {
		result := resolveFolderPath(mockFS, "nonexistent/path/file.mp3")
		Expect(result).To(Equal(""))
	})

	It("returns empty string for dot path", func() {
		result := resolveFolderPath(mockFS, ".")
		Expect(result).To(Equal(""))
	})

	It("returns empty string for empty path", func() {
		result := resolveFolderPath(mockFS, "")
		Expect(result).To(Equal(""))
	})

	It("handles nested file paths correctly", func() {
		result := resolveFolderPath(mockFS, "artist1/album2/song.flac")
		Expect(result).To(Equal("artist1/album2"))
	})

	It("resolves to top-level directory", func() {
		result := resolveFolderPath(mockFS, "artist2/cover.jpg")
		Expect(result).To(Equal("artist2"))
	})
})
