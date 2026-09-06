package scanner

import (
	"context"
	"io/fs"

	"github.com/navidrome/navidrome/core/scannerworker"
	"github.com/navidrome/navidrome/log"
)

func (f *folderEntry) hash() string {
	if f.rustHash != "" {
		return f.rustHash
	}
	if f.job.localRoot != "" {
		if hash, err := rustFolderHash(f); err == nil && hash != "" {
			f.rustHash = hash
			return f.rustHash
		} else if err != nil {
			log.Warn("Scanner: Rust folder hash failed", "path", f.path, err)
		} else {
			log.Warn("Scanner: folder hash missing from Rust scanner on local library path", "path", f.path)
		}
		log.Error("Scanner: not using Go MD5 for local library folder hash", "path", f.path)
		return ""
	}
	return f.hashGo()
}

func rustFolderHash(f *folderEntry) (string, error) {
	request := scannerworker.FolderHashRequest{
		Path:              f.path,
		ModTimeNS:         f.modTime.UnixNano(),
		ImagesUpdatedAtNS: f.imagesUpdatedAt.UnixNano(),
		NumPlaylists:      f.numPlaylists,
		NumSubfolders:     f.numSubFolders,
		AudioFiles:        folderHashFiles(f.audioFiles, f.fileInfos),
		ImageFiles:        folderHashFiles(f.imageFiles, f.fileInfos),
	}
	return scannerworker.PersistentFolderHashWorkers().Hash(context.Background(), request)
}

func folderHashFiles(entries map[string]fs.DirEntry, infos map[string]fs.FileInfo) map[string]scannerworker.FolderHashFile {
	if len(entries) == 0 {
		return nil
	}
	out := make(map[string]scannerworker.FolderHashFile, len(entries))
	for name, entry := range entries {
		info, err := fileInfoForHash(name, entry, infos)
		if err != nil {
			continue
		}
		out[name] = scannerworker.FolderHashFile{
			Name:      name,
			Size:      uint64(info.Size()),
			ModTimeNS: info.ModTime().UnixNano(),
		}
	}
	return out
}

func fileInfoForHash(name string, entry fs.DirEntry, infos map[string]fs.FileInfo) (fs.FileInfo, error) {
	if info, ok := infos[name]; ok {
		return info, nil
	}
	return entry.Info()
}
