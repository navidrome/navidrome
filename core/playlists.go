package core

import (
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/RaveNoX/go-jsoncommentstrip"
	"github.com/bmatcuk/doublestar/v4"
	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/criteria"
	"github.com/navidrome/navidrome/model/request"
	"github.com/navidrome/navidrome/utils/ioutils"
	"github.com/navidrome/navidrome/utils/slice"
	"golang.org/x/text/unicode/norm"
)

type Playlists interface {
	ImportFile(ctx context.Context, folder *model.Folder, filename string) (*model.Playlist, error)
	Update(ctx context.Context, playlistID string, name *string, comment *string, public *bool, idsToAdd []string, idxToRemove []int) error
	ImportM3U(ctx context.Context, reader io.Reader) (*model.Playlist, error)
}

type playlists struct {
	ds model.DataStore
}

func NewPlaylists(ds model.DataStore) Playlists {
	return &playlists{ds: ds}
}

func InPlaylistsPath(folder model.Folder) bool {
	if conf.Server.PlaylistsPath == "" {
		return true
	}
	rel, _ := filepath.Rel(folder.LibraryPath, folder.AbsolutePath())
	for _, path := range strings.Split(conf.Server.PlaylistsPath, string(filepath.ListSeparator)) {
		if match, _ := doublestar.Match(path, rel); match {
			return true
		}
	}
	return false
}

func (s *playlists) ImportFile(ctx context.Context, folder *model.Folder, filename string) (*model.Playlist, error) {
	pls, err := s.parsePlaylist(ctx, filename, folder)
	if err != nil {
		log.Error(ctx, "Error parsing playlist", "path", filepath.Join(folder.AbsolutePath(), filename), err)
		return nil, err
	}
	log.Debug("Found playlist", "name", pls.Name, "lastUpdated", pls.UpdatedAt, "path", pls.Path, "numTracks", len(pls.Tracks))
	err = s.updatePlaylist(ctx, pls)
	if err != nil {
		log.Error(ctx, "Error updating playlist", "path", filepath.Join(folder.AbsolutePath(), filename), err)
	}
	return pls, err
}

func (s *playlists) ImportM3U(ctx context.Context, reader io.Reader) (*model.Playlist, error) {
	owner, _ := request.UserFrom(ctx)
	pls := &model.Playlist{
		OwnerID: owner.ID,
		Public:  false,
		Sync:    false,
	}
	err := s.parseM3U(ctx, pls, nil, reader)
	if err != nil {
		log.Error(ctx, "Error parsing playlist", err)
		return nil, err
	}
	err = s.ds.Playlist(ctx).Put(pls)
	if err != nil {
		log.Error(ctx, "Error saving playlist", err)
		return nil, err
	}
	return pls, nil
}

func (s *playlists) parsePlaylist(ctx context.Context, playlistFile string, folder *model.Folder) (*model.Playlist, error) {
	pls, err := s.newSyncedPlaylist(folder.AbsolutePath(), playlistFile)
	if err != nil {
		return nil, err
	}

	file, err := os.Open(pls.Path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	reader := ioutils.UTF8Reader(file)
	extension := strings.ToLower(filepath.Ext(playlistFile))
	switch extension {
	case ".nsp":
		err = s.parseNSP(ctx, pls, reader)
	default:
		err = s.parseM3U(ctx, pls, folder, reader)
	}
	return pls, err
}

func (s *playlists) newSyncedPlaylist(baseDir string, playlistFile string) (*model.Playlist, error) {
	playlistPath := filepath.Join(baseDir, playlistFile)
	info, err := os.Stat(playlistPath)
	if err != nil {
		return nil, err
	}

	var extension = filepath.Ext(playlistFile)
	var name = playlistFile[0 : len(playlistFile)-len(extension)]

	pls := &model.Playlist{
		Name:      name,
		Comment:   fmt.Sprintf("Auto-imported from '%s'", playlistFile),
		Public:    false,
		Path:      playlistPath,
		Sync:      true,
		UpdatedAt: info.ModTime(),
	}
	return pls, nil
}

func getPositionFromOffset(data []byte, offset int64) (line, column int) {
	line = 1
	for _, b := range data[:offset] {
		if b == '\n' {
			line++
			column = 1
		} else {
			column++
		}
	}
	return
}

func (s *playlists) parseNSP(_ context.Context, pls *model.Playlist, reader io.Reader) error {
	nsp := &nspFile{}
	reader = io.LimitReader(reader, 100*1024) // Limit to 100KB
	reader = jsoncommentstrip.NewReader(reader)
	input, err := io.ReadAll(reader)
	if err != nil {
		return fmt.Errorf("reading SmartPlaylist: %w", err)
	}
	err = json.Unmarshal(input, nsp)
	if err != nil {
		var syntaxErr *json.SyntaxError
		if errors.As(err, &syntaxErr) {
			line, col := getPositionFromOffset(input, syntaxErr.Offset)
			return fmt.Errorf("JSON syntax error in SmartPlaylist at line %d, column %d: %w", line, col, err)
		}
		return fmt.Errorf("JSON parsing error in SmartPlaylist: %w", err)
	}
	pls.Rules = &nsp.Criteria
	if nsp.Name != "" {
		pls.Name = nsp.Name
	}
	if nsp.Comment != "" {
		pls.Comment = nsp.Comment
	}
	if nsp.Public != nil {
		pls.Public = *nsp.Public
	} else {
		pls.Public = conf.Server.DefaultPlaylistPublicVisibility
	}
	return nil
}

func (s *playlists) parseM3U(ctx context.Context, pls *model.Playlist, folder *model.Folder, reader io.Reader) error {
	mediaFileRepository := s.ds.MediaFile(ctx)
	var mfs model.MediaFiles
	for lines := range slice.CollectChunks(slice.LinesFrom(reader), 400) {
		filteredLines := make([]string, 0, len(lines))
		for _, line := range lines {
			line := strings.TrimSpace(line)
			if strings.HasPrefix(line, "#PLAYLIST:") {
				pls.Name = line[len("#PLAYLIST:"):]
				continue
			}
			// Skip empty lines and extended info
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			if strings.HasPrefix(line, "file://") {
				line = strings.TrimPrefix(line, "file://")
				line, _ = url.QueryUnescape(line)
			}
			if !model.IsAudioFile(line) {
				continue
			}
			filteredLines = append(filteredLines, line)
		}
		resolvedPaths, err := s.resolvePaths(ctx, folder, filteredLines)
		if err != nil {
			log.Warn(ctx, "Error resolving paths in playlist", "playlist", pls.Name, err)
			continue
		}

		// Normalize to NFD for filesystem compatibility (macOS). Database stores paths in NFD.
		// See https://github.com/navidrome/navidrome/issues/4663
		resolvedPaths = slice.Map(resolvedPaths, func(path string) string {
			return strings.ToLower(norm.NFD.String(path))
		})

		found, err := mediaFileRepository.FindByPaths(resolvedPaths)
		if err != nil {
			log.Warn(ctx, "Error reading files from DB", "playlist", pls.Name, err)
			continue
		}
		// Build lookup map with library-qualified keys, normalized for comparison
		existing := make(map[string]int, len(found))
		for idx := range found {
			// Normalize to lowercase for case-insensitive comparison
			// Key format: "libraryID:path"
			key := fmt.Sprintf("%d:%s", found[idx].LibraryID, strings.ToLower(found[idx].Path))
			existing[key] = idx
		}

		// Find media files in the order of the resolved paths, to keep playlist order
		for _, path := range resolvedPaths {
			idx, ok := existing[path]
			if ok {
				mfs = append(mfs, found[idx])
			} else {
				log.Warn(ctx, "Path in playlist not found", "playlist", pls.Name, "path", path)
			}
		}
	}
	if pls.Name == "" {
		pls.Name = time.Now().Format(time.RFC3339)
	}
	pls.Tracks = nil
	pls.AddMediaFiles(mfs)

	return nil
}

// pathResolution holds the result of resolving a playlist path to a library-relative path.
type pathResolution struct {
	absolutePath string
	libraryPath  string
	libraryID    int
	valid        bool
}

// ToQualifiedString converts the path resolution to a library-qualified string with forward slashes.
// Format: "libraryID:relativePath" with forward slashes for path separators.
func (r pathResolution) ToQualifiedString() (string, error) {
	if !r.valid {
		return "", fmt.Errorf("invalid path resolution")
	}
	relativePath, err := filepath.Rel(r.libraryPath, r.absolutePath)
	if err != nil {
		return "", err
	}
	// Convert path separators to forward slashes
	return fmt.Sprintf("%d:%s", r.libraryID, filepath.ToSlash(relativePath)), nil
}

// libraryMatcher holds sorted libraries with cleaned paths for efficient path matching.
type libraryMatcher struct {
	libraries    model.Libraries
	cleanedPaths []string
}

// findLibraryForPath finds which library contains the given absolute path.
// Returns library ID and path, or 0 and empty string if not found.
func (lm *libraryMatcher) findLibraryForPath(absolutePath string) (int, string) {
	// Check sorted libraries (longest path first) to find the best match
	for i, cleanLibPath := range lm.cleanedPaths {
		// Check if absolutePath is under this library path
		if strings.HasPrefix(absolutePath, cleanLibPath) {
			// Ensure it's a proper path boundary (not just a prefix)
			if len(absolutePath) == len(cleanLibPath) || absolutePath[len(cleanLibPath)] == filepath.Separator {
				return lm.libraries[i].ID, cleanLibPath
			}
		}
	}
	return 0, ""
}

// newLibraryMatcher creates a libraryMatcher with libraries sorted by path length (longest first).
// This ensures correct matching when library paths are prefixes of each other.
// Example: /music-classical must be checked before /music
// Otherwise, /music-classical/track.mp3 would match /music instead of /music-classical
func newLibraryMatcher(libs model.Libraries) *libraryMatcher {
	// Sort libraries by path length (descending) to ensure longest paths match first.
	slices.SortFunc(libs, func(i, j model.Library) int {
		return cmp.Compare(len(j.Path), len(i.Path)) // Reverse order for descending
	})

	// Pre-clean all library paths once for efficient matching
	cleanedPaths := make([]string, len(libs))
	for i, lib := range libs {
		cleanedPaths[i] = filepath.Clean(lib.Path)
	}
	return &libraryMatcher{
		libraries:    libs,
		cleanedPaths: cleanedPaths,
	}
}

// pathResolver handles path resolution logic for playlist imports.
type pathResolver struct {
	matcher *libraryMatcher
}

// newPathResolver creates a pathResolver with libraries loaded from the datastore.
func newPathResolver(ctx context.Context, ds model.DataStore) (*pathResolver, error) {
	libs, err := ds.Library(ctx).GetAll()
	if err != nil {
		return nil, err
	}
	matcher := newLibraryMatcher(libs)
	return &pathResolver{matcher: matcher}, nil
}

// resolvePath determines the absolute path and library path for a playlist entry.
// For absolute paths, it uses them directly.
// For relative paths, it resolves them relative to the playlist's folder location.
// Example: playlist at /music/playlists/test.m3u with line "../songs/abc.mp3"
//
//	resolves to /music/songs/abc.mp3
func (r *pathResolver) resolvePath(line string, folder *model.Folder) pathResolution {
	var absolutePath string
	if folder != nil && !filepath.IsAbs(line) {
		// Resolve relative path to absolute path based on playlist location
		absolutePath = filepath.Clean(filepath.Join(folder.AbsolutePath(), line))
	} else {
		// Use absolute path directly after cleaning
		absolutePath = filepath.Clean(line)
	}

	return r.findInLibraries(absolutePath)
}

// findInLibraries matches an absolute path against all known libraries and returns
// a pathResolution with the library information. Returns an invalid resolution if
// the path is not found in any library.
func (r *pathResolver) findInLibraries(absolutePath string) pathResolution {
	libID, libPath := r.matcher.findLibraryForPath(absolutePath)
	if libID == 0 {
		return pathResolution{valid: false}
	}
	return pathResolution{
		absolutePath: absolutePath,
		libraryPath:  libPath,
		libraryID:    libID,
		valid:        true,
	}
}

// resolvePaths converts playlist file paths to library-qualified paths (format: "libraryID:relativePath").
// For relative paths, it resolves them to absolute paths first, then determines which
// library they belong to. This allows playlists to reference files across library boundaries.
func (s *playlists) resolvePaths(ctx context.Context, folder *model.Folder, lines []string) ([]string, error) {
	resolver, err := newPathResolver(ctx, s.ds)
	if err != nil {
		return nil, err
	}

	results := make([]string, 0, len(lines))
	for idx, line := range lines {
		resolution := resolver.resolvePath(line, folder)

		if !resolution.valid {
			log.Warn(ctx, "Path in playlist not found in any library", "path", line, "line", idx)
			continue
		}

		qualifiedPath, err := resolution.ToQualifiedString()
		if err != nil {
			log.Debug(ctx, "Error getting library-qualified path", "path", line,
				"libPath", resolution.libraryPath, "filePath", resolution.absolutePath, err)
			continue
		}

		results = append(results, qualifiedPath)
	}

	return results, nil
}

func (s *playlists) updatePlaylist(ctx context.Context, newPls *model.Playlist) error {
	owner, _ := request.UserFrom(ctx)

	pls, err := s.ds.Playlist(ctx).FindByPath(newPls.Path)
	if err != nil && !errors.Is(err, model.ErrNotFound) {
		return err
	}
	if err == nil && !pls.Sync {
		log.Debug(ctx, "Playlist already imported and not synced", "playlist", pls.Name, "path", pls.Path)
		return nil
	}

	if err == nil {
		log.Info(ctx, "Updating synced playlist", "playlist", pls.Name, "path", newPls.Path)
		newPls.ID = pls.ID
		newPls.Name = pls.Name
		newPls.Comment = pls.Comment
		newPls.OwnerID = pls.OwnerID
		newPls.Public = pls.Public
		newPls.EvaluatedAt = &time.Time{}
	} else {
		log.Info(ctx, "Adding synced playlist", "playlist", newPls.Name, "path", newPls.Path, "owner", owner.UserName)
		newPls.OwnerID = owner.ID
		// For NSP files, Public may already be set from the file; for M3U, use server default
		if !newPls.IsSmartPlaylist() {
			newPls.Public = conf.Server.DefaultPlaylistPublicVisibility
		}
	}
	return s.ds.Playlist(ctx).Put(newPls)
}

func (s *playlists) Update(ctx context.Context, playlistID string,
	name *string, comment *string, public *bool,
	idsToAdd []string, idxToRemove []int) error {
	needsInfoUpdate := name != nil || comment != nil || public != nil
	needsTrackRefresh := len(idxToRemove) > 0

	return s.ds.WithTxImmediate(func(tx model.DataStore) error {
		var pls *model.Playlist
		var err error
		repo := tx.Playlist(ctx)
		tracks := repo.Tracks(playlistID, true)
		if tracks == nil {
			return fmt.Errorf("%w: playlist '%s'", model.ErrNotFound, playlistID)
		}
		if needsTrackRefresh {
			pls, err = repo.GetWithTracks(playlistID, true, false)
			pls.RemoveTracks(idxToRemove)
			pls.AddMediaFilesByID(idsToAdd)
		} else {
			if len(idsToAdd) > 0 {
				_, err = tracks.Add(idsToAdd)
				if err != nil {
					return err
				}
			}
			if needsInfoUpdate {
				pls, err = repo.Get(playlistID)
			}
		}
		if err != nil {
			return err
		}
		if !needsTrackRefresh && !needsInfoUpdate {
			return nil
		}

		if name != nil {
			pls.Name = *name
		}
		if comment != nil {
			pls.Comment = *comment
		}
		if public != nil {
			pls.Public = *public
		}
		// Special case: The playlist is now empty
		if len(idxToRemove) > 0 && len(pls.Tracks) == 0 {
			if err = tracks.DeleteAll(); err != nil {
				return err
			}
		}
		return repo.Put(pls)
	})
}

type nspFile struct {
	criteria.Criteria
	Name    string `json:"name"`
	Comment string `json:"comment"`
	Public  *bool  `json:"public"`
}

func (i *nspFile) UnmarshalJSON(data []byte) error {
	m := map[string]interface{}{}
	err := json.Unmarshal(data, &m)
	if err != nil {
		return err
	}
	i.Name, _ = m["name"].(string)
	i.Comment, _ = m["comment"].(string)
	if public, ok := m["public"].(bool); ok {
		i.Public = &public
	}
	return json.Unmarshal(data, &i.Criteria)
}
