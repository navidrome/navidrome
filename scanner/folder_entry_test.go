package scanner

import (
	"io/fs"
	"time"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/conf/configtest"
	"github.com/navidrome/navidrome/model"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("folder_entry", func() {
	var (
		lib  model.Library
		job  *scanJob
		path string
	)

	BeforeEach(func() {
		DeferCleanup(configtest.SetupConfig())
		lib = model.Library{
			ID:                 500,
			Path:               "/music",
			LastScanStartedAt:  time.Now().Add(-1 * time.Hour),
			FullScanInProgress: false,
		}
		job = &scanJob{
			lib:         lib,
			lastUpdates: make(map[string]model.FolderUpdateInfo),
		}
		path = "test/folder"
	})

	Describe("newFolderEntry", func() {
		It("creates a new folder entry with correct initialization", func() {
			folderID := model.FolderID(lib, path)
			updateInfo := model.FolderUpdateInfo{
				UpdatedAt: time.Now().Add(-30 * time.Minute),
				Hash:      "previous-hash",
			}
			job.lastUpdates[folderID] = updateInfo

			entry := newFolderEntry(job, path)

			Expect(entry.id).To(Equal(folderID))
			Expect(entry.job).To(Equal(job))
			Expect(entry.path).To(Equal(path))
			Expect(entry.audioFiles).To(BeEmpty())
			Expect(entry.imageFiles).To(BeEmpty())
			Expect(entry.albumIDMap).To(BeEmpty())
			Expect(entry.updTime).To(Equal(updateInfo.UpdatedAt))
			Expect(entry.prevHash).To(Equal(updateInfo.Hash))
		})

		It("creates a new folder entry with zero time when no previous update exists", func() {
			entry := newFolderEntry(job, path)

			Expect(entry.updTime).To(BeZero())
			Expect(entry.prevHash).To(BeEmpty())
		})

		It("removes the lastUpdate from the job after popping", func() {
			folderID := model.FolderID(lib, path)
			updateInfo := model.FolderUpdateInfo{
				UpdatedAt: time.Now().Add(-30 * time.Minute),
				Hash:      "previous-hash",
			}
			job.lastUpdates[folderID] = updateInfo

			newFolderEntry(job, path)

			Expect(job.lastUpdates).ToNot(HaveKey(folderID))
		})
	})

	Describe("folderEntry", func() {
		var entry *folderEntry

		BeforeEach(func() {
			entry = newFolderEntry(job, path)
		})

		Describe("hasNoFiles", func() {
			It("returns true when folder has no files or subfolders", func() {
				Expect(entry.hasNoFiles()).To(BeTrue())
			})

			It("returns false when folder has audio files", func() {
				entry.audioFiles["test.mp3"] = &fakeDirEntry{name: "test.mp3"}
				Expect(entry.hasNoFiles()).To(BeFalse())
			})

			It("returns false when folder has image files", func() {
				entry.imageFiles["cover.jpg"] = &fakeDirEntry{name: "cover.jpg"}
				Expect(entry.hasNoFiles()).To(BeFalse())
			})

			It("returns false when folder has playlists", func() {
				entry.numPlaylists = 1
				Expect(entry.hasNoFiles()).To(BeFalse())
			})

			It("ignores subfolders when checking for no files", func() {
				entry.numSubFolders = 1
				Expect(entry.hasNoFiles()).To(BeTrue())
			})

			It("returns false when folder has multiple types of content", func() {
				entry.audioFiles["test.mp3"] = &fakeDirEntry{name: "test.mp3"}
				entry.imageFiles["cover.jpg"] = &fakeDirEntry{name: "cover.jpg"}
				entry.numPlaylists = 2
				entry.numSubFolders = 3
				Expect(entry.hasNoFiles()).To(BeFalse())
			})
		})

		Describe("isEmpty", func() {
			It("returns true when folder has no files or subfolders", func() {
				Expect(entry.isEmpty()).To(BeTrue())
			})
			It("returns false when folder has audio files", func() {
				entry.audioFiles["test.mp3"] = &fakeDirEntry{name: "test.mp3"}
				Expect(entry.isEmpty()).To(BeFalse())
			})
			It("returns false when folder has subfolders", func() {
				entry.numSubFolders = 1
				Expect(entry.isEmpty()).To(BeFalse())
			})
		})

		Describe("isNew", func() {
			It("returns true when updTime is zero", func() {
				entry.updTime = time.Time{}
				Expect(entry.isNew()).To(BeTrue())
			})

			It("returns false when updTime is not zero", func() {
				entry.updTime = time.Now()
				Expect(entry.isNew()).To(BeFalse())
			})
		})

		Describe("toFolder", func() {
			BeforeEach(func() {
				entry.audioFiles = map[string]fs.DirEntry{
					"song1.mp3": &fakeDirEntry{name: "song1.mp3"},
					"song2.mp3": &fakeDirEntry{name: "song2.mp3"},
				}
				entry.imageFiles = map[string]fs.DirEntry{
					"cover.jpg":  &fakeDirEntry{name: "cover.jpg"},
					"folder.png": &fakeDirEntry{name: "folder.png"},
				}
				entry.numPlaylists = 3
				entry.imagesUpdatedAt = time.Now()
			})

			It("converts folder entry to model.Folder correctly", func() {
				folder := entry.toFolder()

				Expect(folder.LibraryID).To(Equal(lib.ID))
				Expect(folder.ID).To(Equal(entry.id))
				Expect(folder.NumAudioFiles).To(Equal(2))
				Expect(folder.ImageFiles).To(ConsistOf("cover.jpg", "folder.png"))
				Expect(folder.ImagesUpdatedAt).To(Equal(entry.imagesUpdatedAt))
				Expect(folder.Hash).To(Equal(entry.hash()))
			})

			It("sets NumPlaylists when folder is in playlists path", func() {
				// Mock InPlaylistsPath to return true by setting empty PlaylistsPath
				originalPath := conf.Server.PlaylistsPath
				conf.Server.PlaylistsPath = ""
				DeferCleanup(func() { conf.Server.PlaylistsPath = originalPath })

				folder := entry.toFolder()
				Expect(folder.NumPlaylists).To(Equal(3))
			})

			It("does not set NumPlaylists when folder is not in playlists path", func() {
				// Mock InPlaylistsPath to return false by setting a different path
				originalPath := conf.Server.PlaylistsPath
				conf.Server.PlaylistsPath = "different/path"
				DeferCleanup(func() { conf.Server.PlaylistsPath = originalPath })

				folder := entry.toFolder()
				Expect(folder.NumPlaylists).To(BeZero())
			})
		})

		Describe("hash", func() {
			BeforeEach(func() {
				entry.modTime = time.Date(2023, 1, 15, 12, 0, 0, 0, time.UTC)
				entry.imagesUpdatedAt = time.Date(2023, 1, 16, 14, 30, 0, 0, time.UTC)
			})

			It("produces deterministic hash for same content", func() {
				entry.audioFiles = map[string]fs.DirEntry{
					"b.mp3": &fakeDirEntry{name: "b.mp3"},
					"a.mp3": &fakeDirEntry{name: "a.mp3"},
				}
				entry.imageFiles = map[string]fs.DirEntry{
					"z.jpg": &fakeDirEntry{name: "z.jpg"},
					"x.png": &fakeDirEntry{name: "x.png"},
				}
				entry.numPlaylists = 2
				entry.numSubFolders = 3

				hash1 := entry.hash()

				// Reverse order of maps
				entry.audioFiles = map[string]fs.DirEntry{
					"a.mp3": &fakeDirEntry{name: "a.mp3"},
					"b.mp3": &fakeDirEntry{name: "b.mp3"},
				}
				entry.imageFiles = map[string]fs.DirEntry{
					"x.png": &fakeDirEntry{name: "x.png"},
					"z.jpg": &fakeDirEntry{name: "z.jpg"},
				}

				hash2 := entry.hash()
				Expect(hash1).To(Equal(hash2))
			})

			It("produces different hash when audio files change", func() {
				entry.audioFiles = map[string]fs.DirEntry{
					"song1.mp3": &fakeDirEntry{name: "song1.mp3"},
				}
				hash1 := entry.hash()

				entry.audioFiles["song2.mp3"] = &fakeDirEntry{name: "song2.mp3"}
				hash2 := entry.hash()

				Expect(hash1).ToNot(Equal(hash2))
			})

			It("produces different hash when image files change", func() {
				entry.imageFiles = map[string]fs.DirEntry{
					"cover.jpg": &fakeDirEntry{name: "cover.jpg"},
				}
				hash1 := entry.hash()

				entry.imageFiles["folder.png"] = &fakeDirEntry{name: "folder.png"}
				hash2 := entry.hash()

				Expect(hash1).ToNot(Equal(hash2))
			})

			It("produces different hash when modification time changes", func() {
				hash1 := entry.hash()

				entry.modTime = entry.modTime.Add(1 * time.Hour)
				hash2 := entry.hash()

				Expect(hash1).ToNot(Equal(hash2))
			})

			It("produces different hash when playlist count changes", func() {
				hash1 := entry.hash()

				entry.numPlaylists = 5
				hash2 := entry.hash()

				Expect(hash1).ToNot(Equal(hash2))
			})

			It("produces different hash when subfolder count changes", func() {
				hash1 := entry.hash()

				entry.numSubFolders = 3
				hash2 := entry.hash()

				Expect(hash1).ToNot(Equal(hash2))
			})

			It("produces different hash when images updated time changes", func() {
				hash1 := entry.hash()

				entry.imagesUpdatedAt = entry.imagesUpdatedAt.Add(2 * time.Hour)
				hash2 := entry.hash()

				Expect(hash1).ToNot(Equal(hash2))
			})

			It("produces different hash when audio file size changes", func() {
				entry.audioFiles["test.mp3"] = &fakeDirEntry{
					name: "test.mp3",
					fileInfo: &fakeFileInfo{
						name:    "test.mp3",
						size:    1000,
						modTime: time.Now(),
					},
				}
				hash1 := entry.hash()

				entry.audioFiles["test.mp3"] = &fakeDirEntry{
					name: "test.mp3",
					fileInfo: &fakeFileInfo{
						name:    "test.mp3",
						size:    2000, // Different size
						modTime: time.Now(),
					},
				}
				hash2 := entry.hash()

				Expect(hash1).ToNot(Equal(hash2))
			})

			It("produces different hash when audio file modification time changes", func() {
				baseTime := time.Now()
				entry.audioFiles["test.mp3"] = &fakeDirEntry{
					name: "test.mp3",
					fileInfo: &fakeFileInfo{
						name:    "test.mp3",
						size:    1000,
						modTime: baseTime,
					},
				}
				hash1 := entry.hash()

				entry.audioFiles["test.mp3"] = &fakeDirEntry{
					name: "test.mp3",
					fileInfo: &fakeFileInfo{
						name:    "test.mp3",
						size:    1000,
						modTime: baseTime.Add(1 * time.Hour), // Different modtime
					},
				}
				hash2 := entry.hash()

				Expect(hash1).ToNot(Equal(hash2))
			})

			It("produces different hash when image file size changes", func() {
				entry.imageFiles["cover.jpg"] = &fakeDirEntry{
					name: "cover.jpg",
					fileInfo: &fakeFileInfo{
						name:    "cover.jpg",
						size:    5000,
						modTime: time.Now(),
					},
				}
				hash1 := entry.hash()

				entry.imageFiles["cover.jpg"] = &fakeDirEntry{
					name: "cover.jpg",
					fileInfo: &fakeFileInfo{
						name:    "cover.jpg",
						size:    6000, // Different size
						modTime: time.Now(),
					},
				}
				hash2 := entry.hash()

				Expect(hash1).ToNot(Equal(hash2))
			})

			It("produces different hash when image file modification time changes", func() {
				baseTime := time.Now()
				entry.imageFiles["cover.jpg"] = &fakeDirEntry{
					name: "cover.jpg",
					fileInfo: &fakeFileInfo{
						name:    "cover.jpg",
						size:    5000,
						modTime: baseTime,
					},
				}
				hash1 := entry.hash()

				entry.imageFiles["cover.jpg"] = &fakeDirEntry{
					name: "cover.jpg",
					fileInfo: &fakeFileInfo{
						name:    "cover.jpg",
						size:    5000,
						modTime: baseTime.Add(1 * time.Hour), // Different modtime
					},
				}
				hash2 := entry.hash()

				Expect(hash1).ToNot(Equal(hash2))
			})

			It("produces valid hex-encoded hash", func() {
				hash := entry.hash()
				Expect(hash).To(HaveLen(32)) // MD5 hash should be 32 hex characters
				Expect(hash).To(MatchRegexp("^[a-f0-9]{32}$"))
			})
		})

		Describe("isOutdated", func() {
			BeforeEach(func() {
				entry.prevHash = entry.hash()
			})

			Context("when full scan is in progress", func() {
				BeforeEach(func() {
					entry.job.lib.FullScanInProgress = true
					entry.job.lib.LastScanStartedAt = time.Now()
				})

				It("returns true when updTime is before LastScanStartedAt", func() {
					entry.updTime = entry.job.lib.LastScanStartedAt.Add(-1 * time.Hour)
					Expect(entry.isOutdated()).To(BeTrue())
				})

				It("returns false when updTime is after LastScanStartedAt", func() {
					entry.updTime = entry.job.lib.LastScanStartedAt.Add(1 * time.Hour)
					Expect(entry.isOutdated()).To(BeFalse())
				})

				It("returns false when updTime equals LastScanStartedAt", func() {
					entry.updTime = entry.job.lib.LastScanStartedAt
					Expect(entry.isOutdated()).To(BeFalse())
				})
			})

			Context("when full scan is not in progress", func() {
				BeforeEach(func() {
					entry.job.lib.FullScanInProgress = false
				})

				It("returns false when hash hasn't changed", func() {
					Expect(entry.isOutdated()).To(BeFalse())
				})

				It("returns true when hash has changed", func() {
					entry.numPlaylists = 10 // Change something to change the hash
					Expect(entry.isOutdated()).To(BeTrue())
				})

				It("returns true when prevHash is empty", func() {
					entry.prevHash = ""
					Expect(entry.isOutdated()).To(BeTrue())
				})
			})

			Context("priority between conditions", func() {
				BeforeEach(func() {
					entry.job.lib.FullScanInProgress = true
					entry.job.lib.LastScanStartedAt = time.Now()
					entry.updTime = entry.job.lib.LastScanStartedAt.Add(-1 * time.Hour)
				})

				It("returns true for full scan condition even when hash hasn't changed", func() {
					// Hash is the same but full scan condition should take priority
					Expect(entry.isOutdated()).To(BeTrue())
				})

				It("returns true when full scan condition is not met but hash changed", func() {
					entry.updTime = entry.job.lib.LastScanStartedAt.Add(1 * time.Hour)
					entry.numPlaylists = 10 // Change hash
					Expect(entry.isOutdated()).To(BeTrue())
				})
			})
		})
	})

	Describe("integration scenarios", func() {
		It("handles complete folder lifecycle", func() {
			// Create new folder entry
			entry := newFolderEntry(job, "music/rock/album")

			// Initially new and has no files
			Expect(entry.isNew()).To(BeTrue())
			Expect(entry.hasNoFiles()).To(BeTrue())

			// Add some files
			entry.audioFiles["track1.mp3"] = &fakeDirEntry{name: "track1.mp3"}
			entry.audioFiles["track2.mp3"] = &fakeDirEntry{name: "track2.mp3"}
			entry.imageFiles["cover.jpg"] = &fakeDirEntry{name: "cover.jpg"}
			entry.numSubFolders = 1
			entry.modTime = time.Now()
			entry.imagesUpdatedAt = time.Now()

			// No longer empty
			Expect(entry.hasNoFiles()).To(BeFalse())

			// Set previous hash to current hash (simulating it's been saved)
			entry.prevHash = entry.hash()
			entry.updTime = time.Now()

			// Should not be new or outdated
			Expect(entry.isNew()).To(BeFalse())
			Expect(entry.isOutdated()).To(BeFalse())

			// Convert to model folder
			folder := entry.toFolder()
			Expect(folder.NumAudioFiles).To(Equal(2))
			Expect(folder.ImageFiles).To(HaveLen(1))
			Expect(folder.Hash).To(Equal(entry.hash()))

			// Modify folder and verify it becomes outdated
			entry.audioFiles["track3.mp3"] = &fakeDirEntry{name: "track3.mp3"}
			Expect(entry.isOutdated()).To(BeTrue())
		})
	})
})

// fakeDirEntry implements fs.DirEntry for testing
type fakeDirEntry struct {
	name     string
	isDir    bool
	typ      fs.FileMode
	fileInfo fs.FileInfo
}

func (f *fakeDirEntry) Name() string {
	return f.name
}

func (f *fakeDirEntry) IsDir() bool {
	return f.isDir
}

func (f *fakeDirEntry) Type() fs.FileMode {
	return f.typ
}

func (f *fakeDirEntry) Info() (fs.FileInfo, error) {
	if f.fileInfo != nil {
		return f.fileInfo, nil
	}
	return &fakeFileInfo{
		name:  f.name,
		isDir: f.isDir,
		mode:  f.typ,
	}, nil
}

// fakeFileInfo implements fs.FileInfo for testing
type fakeFileInfo struct {
	name    string
	size    int64
	mode    fs.FileMode
	modTime time.Time
	isDir   bool
}

func (f *fakeFileInfo) Name() string       { return f.name }
func (f *fakeFileInfo) Size() int64        { return f.size }
func (f *fakeFileInfo) Mode() fs.FileMode  { return f.mode }
func (f *fakeFileInfo) ModTime() time.Time { return f.modTime }
func (f *fakeFileInfo) IsDir() bool        { return f.isDir }
func (f *fakeFileInfo) Sys() any           { return nil }
