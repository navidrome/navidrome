package scannerworker

import (
	"context"
)

type FolderHashFile struct {
	Name      string `json:"name,omitempty"`
	Size      uint64 `json:"size"`
	ModTimeNS int64  `json:"mod_time_ns"`
}

// FolderHashRequest mirrors the Rust FolderHashInput payload.
type FolderHashRequest struct {
	Path              string                    `json:"path"`
	ModTimeNS         int64                     `json:"mod_time_ns"`
	ImagesUpdatedAtNS int64                     `json:"images_updated_at_ns"`
	NumPlaylists      int                       `json:"num_playlists"`
	NumSubfolders     int                       `json:"num_subfolders"`
	AudioFiles        map[string]FolderHashFile `json:"audio_files"`
	ImageFiles        map[string]FolderHashFile `json:"image_files"`
}

type folderHashWorkerPool struct{}

var persistentFolderHashWorkers = &folderHashWorkerPool{}

// PersistentFolderHashWorkers returns the shared folder-hash entrypoint.
func PersistentFolderHashWorkers() *folderHashWorkerPool {
	return persistentFolderHashWorkers
}

func (p *folderHashWorkerPool) Hash(ctx context.Context, request FolderHashRequest) (string, error) {
	return hashGRPC(ctx, request)
}
