package scanner

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"math"
	"time"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/core/rustworker"
	"github.com/navidrome/navidrome/core/scannerworker"
	"github.com/navidrome/navidrome/core/scannerworker/gen"
	"github.com/navidrome/navidrome/log"
)

const (
	maxRustScanEntries = 10_000_000
)

type rustScanRequest struct {
	Root             string            `json:"root"`
	Targets          []string          `json:"targets"`
	FollowSymlinks   bool              `json:"follow_symlinks"`
	IgnoreDotFolders bool              `json:"ignore_dot_folders"`
	KnownHashes      map[string]string `json:"known_hashes,omitempty"`
	WalkThreads      int               `json:"walk_threads,omitempty"`
}

type rustScanFolder struct {
	Path              string                  `json:"path"`
	ModTimeNS         int64                   `json:"mod_time_ns"`
	ImagesUpdatedAtNS int64                   `json:"images_updated_at_ns"`
	NumPlaylists      int                     `json:"num_playlists"`
	NumSubfolders     int                     `json:"num_subfolders"`
	AudioFiles        map[string]rustScanFile `json:"audio_files"`
	ImageFiles        map[string]rustScanFile `json:"image_files"`
	Hash              string                  `json:"hash,omitempty"`
}

type rustScanFile struct {
	Name      string `json:"name"`
	Size      uint64 `json:"size"`
	ModTimeNS int64  `json:"mod_time_ns"`
}

func streamRustFolders(ctx context.Context, job *scanJob, targets []string) (<-chan *rustScanFolder, <-chan error) {
	folders := make(chan *rustScanFolder, 64)
	errs := make(chan error, 1)

	go func() {
		defer close(folders)
		defer close(errs)

		request := rustScanRequest{
			Root:             job.localRoot,
			Targets:          append([]string{}, targets...),
			FollowSymlinks:   conf.Server.Scanner.FollowSymlinks,
			IgnoreDotFolders: conf.Server.Scanner.IgnoreDotFolders,
			KnownHashes:      job.knownHashesSnapshot(),
			WalkThreads:      rustWalkThreads(),
		}

		warn, err := streamRustFoldersGRPC(ctx, request, folders)
		for _, warning := range warn {
			log.Warn(ctx, "Rust scanner traversal warning", "warning", warning)
		}
		errs <- err
	}()

	return folders, errs
}

func streamRustFoldersGRPC(ctx context.Context, request rustScanRequest, folders chan<- *rustScanFolder) ([]string, error) {
	var lastErr error
	for attempt := 0; attempt < 2; attempt++ {
		cli := scannerworker.ScannerGRPC()
		if cli == nil {
			return nil, scannerworker.ErrWalkNoGRPC
		}
		warnings, err := streamRustFoldersGRPCOnce(ctx, cli, request, folders)
		if err == nil {
			return warnings, nil
		}
		lastErr = err
		if !rustworker.IsTransportFailure(err) || attempt > 0 {
			return warnings, err
		}
		scannerworker.InvalidateGRPC()
	}
	return nil, lastErr
}

func streamRustFoldersGRPCOnce(ctx context.Context, cli gen.ScannerClient, request rustScanRequest, folders chan<- *rustScanFolder) ([]string, error) {
	stream, err := cli.Walk(ctx, toProtoWalkRequest(request))
	if err != nil {
		return nil, err
	}

	var warnings []string
	var seen int
	for {
		event, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			return warnings, nil
		}
		if err != nil {
			return warnings, fmt.Errorf("receiving Rust scanner walk event: %w", err)
		}
		switch event.GetKind() {
		case gen.WalkEventKind_WALK_EVENT_KIND_FOLDER:
			folder := folderFromProto(event.GetFolder())
			if folder == nil || folder.Path == "" {
				return warnings, errors.New("Rust scanner returned an invalid folder event")
			}
			seen++
			if seen > maxRustScanEntries {
				return warnings, errors.New("Rust scanner exceeded folder safety limit")
			}
			if err := sendRustFolder(ctx, folders, folder); err != nil {
				return warnings, err
			}
		case gen.WalkEventKind_WALK_EVENT_KIND_FOLDER_SUMMARY:
			folder := event.GetFolder()
			if folder == nil || folder.GetPath() == "" || folder.GetHash() == "" {
				return warnings, errors.New("Rust scanner returned an invalid folder summary event")
			}
			seen++
			if seen > maxRustScanEntries {
				return warnings, errors.New("Rust scanner exceeded folder safety limit")
			}
			if err := sendRustFolder(ctx, folders, &rustScanFolder{
				Path: folder.GetPath(),
				Hash: folder.GetHash(),
			}); err != nil {
				return warnings, err
			}
		case gen.WalkEventKind_WALK_EVENT_KIND_WARNING:
			if event.GetMessage() != "" {
				warnings = append(warnings, event.GetMessage())
			}
		case gen.WalkEventKind_WALK_EVENT_KIND_ERROR:
			msg := event.GetMessage()
			if msg == "" {
				msg = "Rust scanner request failed"
			}
			return warnings, errors.New(msg)
		case gen.WalkEventKind_WALK_EVENT_KIND_DONE:
			return warnings, nil
		default:
			return warnings, fmt.Errorf("Rust scanner returned unknown event %q", event.GetKind())
		}
	}
}

func toProtoWalkRequest(request rustScanRequest) *gen.WalkRequest {
	return &gen.WalkRequest{
		Root:             request.Root,
		Targets:          append([]string{}, request.Targets...),
		FollowSymlinks:   request.FollowSymlinks,
		IgnoreDotFolders: request.IgnoreDotFolders,
		KnownHashes:      request.KnownHashes,
		WalkThreads:      int32(request.WalkThreads),
	}
}

func folderFromProto(folder *gen.WalkFolder) *rustScanFolder {
	if folder == nil {
		return nil
	}
	return &rustScanFolder{
		Path:              folder.GetPath(),
		ModTimeNS:         folder.GetModTimeNs(),
		ImagesUpdatedAtNS: folder.GetImagesUpdatedAtNs(),
		NumPlaylists:      int(folder.GetNumPlaylists()),
		NumSubfolders:     int(folder.GetNumSubfolders()),
		AudioFiles:        filesFromProto(folder.GetAudioFiles()),
		ImageFiles:        filesFromProto(folder.GetImageFiles()),
		Hash:              folder.GetHash(),
	}
}

func toProtoWalkFolder(folder *rustScanFolder) *gen.WalkFolder {
	if folder == nil {
		return nil
	}
	return &gen.WalkFolder{
		Path:              folder.Path,
		ModTimeNs:         folder.ModTimeNS,
		ImagesUpdatedAtNs: folder.ImagesUpdatedAtNS,
		NumPlaylists:      int32(folder.NumPlaylists),
		NumSubfolders:     int32(folder.NumSubfolders),
		AudioFiles:        toProtoScanFiles(folder.AudioFiles),
		ImageFiles:        toProtoScanFiles(folder.ImageFiles),
		Hash:              folder.Hash,
	}
}

func toProtoScanFiles(files map[string]rustScanFile) map[string]*gen.FileMeta {
	if len(files) == 0 {
		return nil
	}
	out := make(map[string]*gen.FileMeta, len(files))
	for name, file := range files {
		fileName := file.Name
		if fileName == "" {
			fileName = name
		}
		out[name] = &gen.FileMeta{Name: fileName, Size: file.Size, ModTimeNs: file.ModTimeNS}
	}
	return out
}

func filesFromProto(files map[string]*gen.FileMeta) map[string]rustScanFile {
	if len(files) == 0 {
		return nil
	}
	out := make(map[string]rustScanFile, len(files))
	for name, file := range files {
		if file == nil {
			continue
		}
		fileName := file.GetName()
		if fileName == "" {
			fileName = name
		}
		out[name] = rustScanFile{Name: fileName, Size: file.GetSize(), ModTimeNS: file.GetModTimeNs()}
	}
	return out
}

func rustWalkThreads() int {
	// Rust streams folders in post-order when walk_threads <= 1. Folder processing
	// stays parallel on the Go side via DevScannerThreads; a parallel Rust walk
	// would buffer the entire tree before the first folder reaches Go.
	return 1
}

func (j *scanJob) knownHashesSnapshot() map[string]string {
	j.lock.Lock()
	defer j.lock.Unlock()
	if len(j.knownHashes) == 0 {
		return nil
	}
	out := make(map[string]string, len(j.knownHashes))
	for path, hash := range j.knownHashes {
		if hash != "" {
			out[path] = hash
		}
	}
	return out
}

func sendRustFolder(ctx context.Context, folders chan<- *rustScanFolder, folder *rustScanFolder) error {
	select {
	case folders <- folder:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func validateRustFolder(folder *rustScanFolder) error {
	for name, file := range folder.AudioFiles {
		if file.Size > math.MaxInt64 {
			return fmt.Errorf("audio file %q in %q exceeds supported size", name, folder.Path)
		}
	}
	for name, file := range folder.ImageFiles {
		if file.Size > math.MaxInt64 {
			return fmt.Errorf("image file %q in %q exceeds supported size", name, folder.Path)
		}
	}
	return nil
}

func folderEntryFromRust(job *scanJob, source *rustScanFolder) (*folderEntry, error) {
	if err := validateRustFolder(source); err != nil {
		return nil, err
	}
	entry := job.createFolderEntry(source.Path)
	entry.path = source.Path
	entry.modTime = unixNanoTime(source.ModTimeNS)
	entry.imagesUpdatedAt = unixNanoTime(source.ImagesUpdatedAtNS)
	entry.numPlaylists = source.NumPlaylists
	entry.numSubFolders = source.NumSubfolders
	entry.rustHash = source.Hash
	for name, file := range source.AudioFiles {
		dirEntry, err := rustDirEntryFromFile(name, file)
		if err != nil {
			return nil, err
		}
		entry.audioFiles[name] = dirEntry
		entry.fileInfos[name] = dirEntry.info
	}
	for name, file := range source.ImageFiles {
		dirEntry, err := rustDirEntryFromFile(name, file)
		if err != nil {
			return nil, err
		}
		entry.imageFiles[name] = dirEntry
		entry.fileInfos[name] = dirEntry.info
	}
	entry.elapsed.Start()
	return entry, nil
}

func rustDirEntryFromFile(name string, file rustScanFile) (*rustDirEntry, error) {
	if file.Size > math.MaxInt64 {
		return nil, fmt.Errorf("file %q exceeds supported size", name)
	}
	return &rustDirEntry{
		name: name,
		info: rustFileInfo{name: name, size: int64(file.Size), modTime: unixNanoTime(file.ModTimeNS)},
	}, nil
}

func unixNanoTime(value int64) time.Time {
	if value == 0 {
		return time.Time{}
	}
	return time.Unix(0, value)
}

type rustDirEntry struct {
	name string
	info rustFileInfo
}

func (e *rustDirEntry) Name() string               { return e.name }
func (e *rustDirEntry) IsDir() bool                { return false }
func (e *rustDirEntry) Type() fs.FileMode          { return 0 }
func (e *rustDirEntry) Info() (fs.FileInfo, error) { return e.info, nil }

type rustFileInfo struct {
	name    string
	size    int64
	modTime time.Time
}

func (i rustFileInfo) Name() string       { return i.name }
func (i rustFileInfo) Size() int64        { return i.size }
func (i rustFileInfo) Mode() fs.FileMode  { return 0 }
func (i rustFileInfo) ModTime() time.Time { return i.modTime }
func (i rustFileInfo) IsDir() bool        { return false }
func (i rustFileInfo) Sys() any           { return nil }
