package local

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"

	"github.com/djherbis/times"
	"github.com/navidrome/navidrome/model/metadata"
	"go.senan.xyz/taglib"
)

// Overlapping libraries can send the same track to different scan workers.
var commentGenreWriteMu sync.Mutex

func commentGenres(comments, existing []string) []string {
	if len(comments) == 0 {
		return existing
	}
	// Only a complete Rekordbox block is a genre list. Surrounding text may contain URLs or notes.
	_, body, found := strings.Cut(comments[0], "/*")
	body, _, closed := strings.Cut(body, "*/")
	if !found || !closed {
		return existing
	}
	genres := slices.Clone(existing)
	for _, genre := range strings.Split(body, "/") {
		genre = strings.TrimSpace(genre)
		if genre != "" && !slices.ContainsFunc(genres, func(v string) bool { return strings.EqualFold(v, genre) }) {
			genres = append(genres, genre)
		}
	}
	return genres
}

func (lfs *localFS) WriteCommentGenres(name string, info *metadata.Info) error {
	// Limit writeback to the audio formats supported by the genre writer.
	ext := strings.ToLower(filepath.Ext(name))
	switch ext {
	case ".flac", ".mp3", ".wav", ".aiff":
	default:
		return nil
	}

	// Merge comment genres and skip tracks whose genre tags already match.
	genres := commentGenres(info.Tags["comment"], info.Tags["genre"])
	if slices.Equal(genres, info.Tags["genre"]) {
		return nil
	}
	if !fs.ValidPath(name) {
		return fs.ErrInvalid
	}

	commentGenreWriteMu.Lock()
	defer commentGenreWriteMu.Unlock()
	// Scan reads can follow symlinks outside the library. Writeback must stay inside the root.
	root, err := os.OpenRoot(lfs.root)
	if err != nil {
		return err
	}
	defer root.Close()
	name = filepath.FromSlash(name)
	// Check the directory entry directly so symlinked tracks are skipped.
	linkInfo, err := root.Lstat(name)
	if err != nil {
		return err
	}
	if !linkInfo.Mode().IsRegular() {
		return nil
	}

	// Check the opened file matches the entry and the scanner's earlier snapshot.
	track, err := root.OpenFile(name, os.O_RDWR, 0)
	if err != nil {
		return err
	}
	defer track.Close()
	original, err := track.Stat()
	if err != nil {
		return err
	}
	if !os.SameFile(linkInfo, original) || original.Size() != info.FileInfo.Size() ||
		!original.ModTime().Equal(info.FileInfo.ModTime()) {
		return errors.New("track changed since its tags were read")
	}

	// Stage and verify the edit before writing it back with rollback available.
	updated, err := writeCommentGenreTags(root, name, track, original, info.Tags, genres)
	if err != nil {
		return err
	}

	// Refresh the scanner's metadata only after writeback succeeds.
	info.FileInfo = localFileInfo{
		FileInfo:    updated,
		path:        filepath.Join(lfs.root, name),
		noBirthTime: &lfs.noBirthTime,
	}
	info.Tags["genre"] = genres
	return nil
}

func writeCommentGenreTags(root *os.Root, name string, track *os.File, original fs.FileInfo, scannedTags map[string][]string, genres []string) (fs.FileInfo, error) {
	// Save and sync a timestamped backup before editing a separate copy.
	// Keep the backup if writeback fails and rollback cannot restore the track.
	scratch, err := os.MkdirTemp("", "navidrome-comment-genres-")
	if err != nil {
		return nil, err
	}
	keepBackup := false
	defer func() {
		if !keepBackup {
			_ = os.RemoveAll(scratch)
		}
	}()
	backup, err := os.Create(filepath.Join(scratch, "original-"+filepath.Base(name)))
	if err != nil {
		return nil, err
	}
	defer backup.Close()
	if err := copyTrack(backup, track); err != nil {
		return nil, fmt.Errorf("saving original track: %w", err)
	}
	originalTimes := times.Get(original)
	if err := os.Chtimes(backup.Name(), originalTimes.AccessTime(), original.ModTime()); err != nil {
		return nil, err
	}
	if err := backup.Sync(); err != nil {
		return nil, err
	}

	staged, err := stageCommentGenreEdit(scratch, filepath.Ext(name), backup, scannedTags, genres)
	if err != nil {
		return nil, err
	}
	defer staged.Close()

	// Preparing the edit can take time. Recheck the library file before writing to it.
	current, err := root.Lstat(name)
	if err != nil {
		return nil, err
	}
	if !os.SameFile(original, current) || current.Size() != original.Size() ||
		!current.ModTime().Equal(original.ModTime()) {
		return nil, errors.New("track changed while preparing genre tags")
	}

	// Keep the original available if the write is interrupted or rollback fails.
	keepBackup = true
	updated, err := replaceGenreTrack(root, name, track, staged, original)
	if err != nil {
		if _, restoreErr := replaceGenreTrack(root, name, track, backup, original); restoreErr != nil {
			return nil, errors.Join(err, fmt.Errorf("restoring track failed; original bytes saved at %q: %w", backup.Name(), restoreErr))
		}
		keepBackup = false
		return nil, err
	}
	keepBackup = false
	return updated, nil
}

func stageCommentGenreEdit(scratch, ext string, backup *os.File, scannedTags map[string][]string, genres []string) (*os.File, error) {
	// TagLib uses the extension to recognize the audio format.
	stagePath := filepath.Join(scratch, "edited"+strings.ToLower(ext))
	staged, err := os.Create(stagePath)
	if err != nil {
		return nil, err
	}
	if err := errors.Join(copyTrack(staged, backup), staged.Close()); err != nil {
		return nil, err
	}
	// File metadata may have changed after the scanner read it. Check the staged input before editing.
	originalTags, err := taglib.ReadTags(stagePath)
	if err != nil {
		return nil, err
	}
	if !slices.Equal(originalTags["COMMENT"], scannedTags["comment"]) ||
		!slices.Equal(originalTags["GENRE"], scannedTags["genre"]) {
		return nil, errors.New("track tags changed since they were read")
	}
	if err := taglib.WriteTags(stagePath, map[string][]string{"GENRE": genres}, 0); err != nil {
		return nil, err
	}
	tags, err := taglib.ReadTags(stagePath)
	if err != nil {
		return nil, err
	}
	// Reject any staged edit that changes other tags exposed by TagLib.
	writtenGenres := tags["GENRE"]
	delete(tags, "GENRE")
	delete(originalTags, "GENRE")
	if !slices.Equal(writtenGenres, genres) || !maps.EqualFunc(tags, originalTags, slices.Equal) {
		return nil, errors.New("staged tag verification failed")
	}
	return os.Open(stagePath)
}

func replaceGenreTrack(root *os.Root, name string, track, src *os.File, original fs.FileInfo) (fs.FileInfo, error) {
	// Rewrite the original inode to preserve its creation time across filesystems.
	if err := copyTrack(track, src); err != nil {
		return nil, err
	}
	current, err := root.Lstat(name)
	if err != nil {
		return nil, err
	}
	if !os.SameFile(original, current) {
		return nil, errors.New("track was replaced during the write")
	}
	// The scanner uses mtime for Recently Added when RecentlyAddedByModTime is enabled.
	if err := root.Chtimes(name, times.Get(original).AccessTime(), original.ModTime()); err != nil {
		return nil, err
	}
	if err := track.Sync(); err != nil {
		return nil, err
	}
	updated, err := track.Stat()
	if err != nil {
		return nil, err
	}
	if !updated.ModTime().Equal(original.ModTime()) {
		return nil, errors.New("file modification time could not be preserved")
	}
	return updated, nil
}

func copyTrack(dst, src *os.File) error {
	if _, err := src.Seek(0, io.SeekStart); err != nil {
		return err
	}
	if _, err := dst.Seek(0, io.SeekStart); err != nil {
		return err
	}
	n, err := io.Copy(dst, src)
	if err != nil {
		return err
	}
	// A shorter edit or rollback must discard leftover bytes from the previous version.
	return dst.Truncate(n)
}
