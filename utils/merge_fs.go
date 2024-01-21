package utils

import (
	"errors"
	"io"
	"io/fs"
	"sort"
)

// MergeFS implements a simple merged fs.FS, that can combine a Base FS with an Overlay FS. The semantics are:
// - Files from the Overlay FS will override files with the same name in the Base FS
// - Directories are combined, with priority for the Overlay FS over the Base FS for files with matching names
type MergeFS struct {
	Base    fs.FS
	Overlay fs.FS
}

func (m MergeFS) Open(name string) (fs.File, error) {
	file, err := m.Overlay.Open(name)
	if err != nil {
		return m.Base.Open(name)
	}

	info, err := file.Stat()
	if err != nil {
		_ = file.Close()
		return nil, err
	}
	overlayDirFile, ok := file.(fs.ReadDirFile)
	if !info.IsDir() || !ok {
		return file, nil
	}

	baseDir, _ := m.Base.Open(name)
	defer func() {
		_ = baseDir.Close()
		_ = file.Close()
	}()
	baseDirFile, ok := baseDir.(fs.ReadDirFile)
	if !ok {
		return nil, fs.ErrInvalid
	}
	return m.mergeDirs(name, info, baseDirFile, overlayDirFile)
}

func (m MergeFS) mergeDirs(name string, info fs.FileInfo, baseDir fs.ReadDirFile, overlayDir fs.ReadDirFile) (fs.File, error) {
	merged := map[string]fs.DirEntry{}

	baseFiles, err := baseDir.ReadDir(-1)
	if err != nil {
		return nil, err
	}
	sort.Slice(baseFiles, func(i, j int) bool { return baseFiles[i].Name() < baseFiles[j].Name() })

	overlayFiles, err := overlayDir.ReadDir(-1)
	if err != nil {
		overlayFiles = nil
	}
	sort.Slice(overlayFiles, func(i, j int) bool { return overlayFiles[i].Name() < overlayFiles[j].Name() })

	for _, f := range baseFiles {
		merged[f.Name()] = f
	}
	for _, f := range overlayFiles {
		merged[f.Name()] = f
	}

	var entries []fs.DirEntry
	for _, i := range merged {
		entries = append(entries, i)
	}

	sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })
	return &mergedDir{
		name:    name,
		info:    info,
		entries: entries,
	}, nil
}

type mergedDir struct {
	name    string
	info    fs.FileInfo
	entries []fs.DirEntry
	pos     int
}

var _ fs.ReadDirFile = (*mergedDir)(nil)

func (d *mergedDir) ReadDir(count int) ([]fs.DirEntry, error) {
	if d.pos >= len(d.entries) && count > 0 {
		return nil, io.EOF
	}
	if count <= 0 || count > len(d.entries)-d.pos {
		count = len(d.entries) - d.pos
	}
	entries := d.entries[d.pos : d.pos+count]
	d.pos += count
	return entries, nil
}

func (d *mergedDir) Close() error               { return nil }
func (d *mergedDir) Stat() (fs.FileInfo, error) { return d.info, nil }
func (d *mergedDir) Read([]byte) (int, error) {
	return 0, &fs.PathError{Op: "read", Path: d.name, Err: errors.New("is a directory")}
}
