package plugins

import (
	"context"
	"io/fs"
	"path/filepath"

	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/tetratelabs/wazero"
	experimentalsys "github.com/tetratelabs/wazero/experimental/sys"
	"github.com/tetratelabs/wazero/experimental/sysfs"
	"github.com/tetratelabs/wazero/sys"
)

// jailedFS confines a mount to its root: wazero resolves guest paths by bare
// concatenation. Named field, not embedding, so a new sys.FS method can't slip through.
type jailedFS struct {
	fs experimentalsys.FS
}

// escapes reports whether a guest path can resolve outside the mount root.
// WASI only validates "/"-separated paths, missing `..\`, `C:x` and device names.
func escapes(path string) bool {
	switch path {
	case "", ".", "/": // how wazero opens the mount root itself
		return false
	}
	return !filepath.IsLocal(path)
}

func (jailedFS) Symlink(string, string) experimentalsys.Errno {
	return experimentalsys.EPERM
}

func (j jailedFS) OpenFile(path string, flag experimentalsys.Oflag, perm fs.FileMode) (experimentalsys.File, experimentalsys.Errno) {
	if escapes(path) {
		return nil, experimentalsys.EPERM
	}
	return j.fs.OpenFile(path, flag, perm)
}

func (j jailedFS) Lstat(path string) (sys.Stat_t, experimentalsys.Errno) {
	if escapes(path) {
		return sys.Stat_t{}, experimentalsys.EPERM
	}
	return j.fs.Lstat(path)
}

func (j jailedFS) Stat(path string) (sys.Stat_t, experimentalsys.Errno) {
	if escapes(path) {
		return sys.Stat_t{}, experimentalsys.EPERM
	}
	return j.fs.Stat(path)
}

func (j jailedFS) Mkdir(path string, perm fs.FileMode) experimentalsys.Errno {
	if escapes(path) {
		return experimentalsys.EPERM
	}
	return j.fs.Mkdir(path, perm)
}

func (j jailedFS) Chmod(path string, perm fs.FileMode) experimentalsys.Errno {
	if escapes(path) {
		return experimentalsys.EPERM
	}
	return j.fs.Chmod(path, perm)
}

func (j jailedFS) Rename(from, to string) experimentalsys.Errno {
	if escapes(from) || escapes(to) {
		return experimentalsys.EPERM
	}
	return j.fs.Rename(from, to)
}

func (j jailedFS) Rmdir(path string) experimentalsys.Errno {
	if escapes(path) {
		return experimentalsys.EPERM
	}
	return j.fs.Rmdir(path)
}

func (j jailedFS) Unlink(path string) experimentalsys.Errno {
	if escapes(path) {
		return experimentalsys.EPERM
	}
	return j.fs.Unlink(path)
}

func (j jailedFS) Link(oldPath, newPath string) experimentalsys.Errno {
	if escapes(oldPath) || escapes(newPath) {
		return experimentalsys.EPERM
	}
	return j.fs.Link(oldPath, newPath)
}

func (j jailedFS) Readlink(path string) (string, experimentalsys.Errno) {
	if escapes(path) {
		return "", experimentalsys.EPERM
	}
	return j.fs.Readlink(path)
}

func (j jailedFS) Utimens(path string, atim, mtim int64) experimentalsys.Errno {
	if escapes(path) {
		return experimentalsys.EPERM
	}
	return j.fs.Utimens(path, atim, mtim)
}

// mount is a host directory exposed to a plugin at guestPath.
type mount struct {
	hostPath  string
	guestPath string
	readOnly  bool
}

// buildMounts lists the libraries the plugin may reach through the filesystem.
func buildMounts(ctx context.Context, libraries model.Libraries, allowedLibraryIDs []int, allLibraries, allowWriteAccess bool) []mount {
	allowedLibrarySet := make(map[int]struct{}, len(allowedLibraryIDs))
	for _, id := range allowedLibraryIDs {
		allowedLibrarySet[id] = struct{}{}
	}
	var mounts []mount
	for _, lib := range libraries {
		_, allowed := allowedLibrarySet[lib.ID]
		if allLibraries || allowed {
			m := mount{hostPath: lib.Path, guestPath: toPluginMountPoint(int32(lib.ID)), readOnly: !allowWriteAccess}
			mounts = append(mounts, m)
			log.Trace(ctx, "Added library to plugin mounts", "libraryID", lib.ID, "mountPoint", m.guestPath, "readOnly", m.readOnly, "hostPath", m.hostPath)
		}
	}
	if allowWriteAccess {
		log.Info(ctx, "Granting read-write filesystem access to libraries", "libraryCount", len(mounts), "allLibraries", allLibraries)
	} else {
		log.Debug(ctx, "Granting read-only filesystem access to libraries", "libraryCount", len(mounts), "allLibraries", allLibraries)
	}
	return mounts
}

// buildFSConfig mounts each host directory jailed to its root.
func buildFSConfig(mounts []mount) wazero.FSConfig {
	if len(mounts) == 0 {
		return nil
	}
	cfg := wazero.NewFSConfig()
	for _, m := range mounts {
		var mounted experimentalsys.FS = jailedFS{fs: sysfs.DirFS(m.hostPath)}
		if m.readOnly {
			mounted = &sysfs.ReadFS{FS: mounted}
		}
		cfg = cfg.(sysfs.FSConfig).WithSysFSMount(mounted, m.guestPath)
	}
	return cfg
}
