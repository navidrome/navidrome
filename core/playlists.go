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
	"github.com/deluan/rest"
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
	// Reads
	GetAll(ctx context.Context, options ...model.QueryOptions) (model.Playlists, error)
	Get(ctx context.Context, id string) (*model.Playlist, error)
	GetWithTracks(ctx context.Context, id string) (*model.Playlist, error)
	GetPlaylists(ctx context.Context, mediaFileId string) (model.Playlists, error)

	// Mutations
	Create(ctx context.Context, playlistId string, name string, ids []string) (string, error)
	Delete(ctx context.Context, id string) error
	Update(ctx context.Context, playlistID string, name *string, comment *string, public *bool, idsToAdd []string, idxToRemove []int) error

	// Track management
	AddTracks(ctx context.Context, playlistID string, ids []string) (int, error)
	AddAlbums(ctx context.Context, playlistID string, albumIds []string) (int, error)
	AddArtists(ctx context.Context, playlistID string, artistIds []string) (int, error)
	AddDiscs(ctx context.Context, playlistID string, discs []model.DiscID) (int, error)
	RemoveTracks(ctx context.Context, playlistID string, trackIds []string) error
	ReorderTrack(ctx context.Context, playlistID string, pos int, newPos int) error

	// Import
	ImportFile(ctx context.Context, folder *model.Folder, filename string) (*model.Playlist, error)
	ImportM3U(ctx context.Context, reader io.Reader) (*model.Playlist, error)

	// REST adapters (follows Share/Library pattern)
	NewRepository(ctx context.Context) rest.Repository
	TracksRepository(ctx context.Context, playlistId string, refreshSmartPlaylist bool) rest.Repository
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
	for path := range strings.SplitSeq(conf.Server.PlaylistsPath, string(filepath.ListSeparator)) {
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
	// Chunk size of 100 lines, as each line can generate up to 4 lookup candidates
	// (NFC/NFD × raw/lowercase), and SQLite has a max expression tree depth of 1000.
	for lines := range slice.CollectChunks(slice.LinesFrom(reader), 100) {
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
			if after, ok := strings.CutPrefix(line, "file://"); ok {
				line = after
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

		// SQLite comparisons do not perform Unicode normalization, and filesystem normalization
		// differs across platforms (macOS often yields NFD, while Linux/Windows typically use NFC).
		// Generate lookup candidates for both forms so playlist entries match DB paths regardless
		// of the original normalization. See https://github.com/navidrome/navidrome/issues/4884
		//
		// We also include the original (non-lowercased) paths because SQLite's COLLATE NOCASE
		// only handles ASCII case-insensitivity. Non-ASCII characters like fullwidth letters
		// (e.g., ＡＢＣＤ vs ａｂｃｄ) are not matched case-insensitively by NOCASE.
		lookupCandidates := make([]string, 0, len(resolvedPaths)*4)
		seen := make(map[string]struct{}, len(resolvedPaths)*4)
		for _, path := range resolvedPaths {
			// Add original paths first (for exact matching of non-ASCII characters)
			nfcRaw := norm.NFC.String(path)
			if _, ok := seen[nfcRaw]; !ok {
				seen[nfcRaw] = struct{}{}
				lookupCandidates = append(lookupCandidates, nfcRaw)
			}
			nfdRaw := norm.NFD.String(path)
			if _, ok := seen[nfdRaw]; !ok {
				seen[nfdRaw] = struct{}{}
				lookupCandidates = append(lookupCandidates, nfdRaw)
			}

			// Add lowercased paths (for ASCII case-insensitive matching via NOCASE)
			nfc := strings.ToLower(nfcRaw)
			if _, ok := seen[nfc]; !ok {
				seen[nfc] = struct{}{}
				lookupCandidates = append(lookupCandidates, nfc)
			}
			nfd := strings.ToLower(nfdRaw)
			if _, ok := seen[nfd]; !ok {
				seen[nfd] = struct{}{}
				lookupCandidates = append(lookupCandidates, nfd)
			}
		}

		found, err := mediaFileRepository.FindByPaths(lookupCandidates)
		if err != nil {
			log.Warn(ctx, "Error reading files from DB", "playlist", pls.Name, err)
			continue
		}

		// Build lookup map with library-qualified keys, normalized for comparison.
		// Canonicalize to NFC so NFD/NFC become comparable.
		existing := make(map[string]int, len(found))
		for idx := range found {
			key := fmt.Sprintf("%d:%s", found[idx].LibraryID, strings.ToLower(norm.NFC.String(found[idx].Path)))
			existing[key] = idx
		}

		// Find media files in the order of the resolved paths, to keep playlist order
		for _, path := range resolvedPaths {
			key := strings.ToLower(norm.NFC.String(path))
			idx, ok := existing[key]
			if ok {
				mfs = append(mfs, found[idx])
			} else {
				// Prefer logging a composed representation when possible to avoid confusing output
				// with decomposed combining marks.
				log.Warn(ctx, "Path in playlist not found", "playlist", pls.Name, "path", norm.NFC.String(path))
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

	// Try to find existing playlist by path. Since filesystem normalization differs across
	// platforms (macOS uses NFD, Linux/Windows use NFC), we try both forms to match
	// playlists that may have been imported on a different platform.
	pls, err := s.ds.Playlist(ctx).FindByPath(newPls.Path)
	if errors.Is(err, model.ErrNotFound) {
		// Try alternate normalization form
		altPath := norm.NFD.String(newPls.Path)
		if altPath == newPls.Path {
			altPath = norm.NFC.String(newPls.Path)
		}
		if altPath != newPls.Path {
			pls, err = s.ds.Playlist(ctx).FindByPath(altPath)
		}
	}
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
	pls, err := s.checkWritable(ctx, playlistID)
	if err != nil {
		return err
	}
	return s.ds.WithTxImmediate(func(tx model.DataStore) error {
		repo := tx.Playlist(ctx)

		if len(idxToRemove) > 0 {
			// Must re-fetch with tracks inside the transaction for index-based removal
			pls, err := repo.GetWithTracks(playlistID, true, false)
			if err != nil {
				return err
			}
			pls.RemoveTracks(idxToRemove)
			pls.AddMediaFilesByID(idsToAdd)
			if len(pls.Tracks) == 0 {
				if err = repo.Tracks(playlistID, false).DeleteAll(); err != nil {
					return err
				}
			}
			return s.updateMetadata(ctx, tx, pls, name, comment, public)
		}

		if len(idsToAdd) > 0 {
			if _, err := repo.Tracks(playlistID, false).Add(idsToAdd); err != nil {
				return err
			}
		}
		if name == nil && comment == nil && public == nil {
			return nil
		}
		// Reuse the playlist from checkWritable (no tracks loaded, so Put only refreshes counters)
		return s.updateMetadata(ctx, tx, pls, name, comment, public)
	})
}

// --- Read operations ---

func (s *playlists) GetAll(ctx context.Context, options ...model.QueryOptions) (model.Playlists, error) {
	return s.ds.Playlist(ctx).GetAll(options...)
}

func (s *playlists) Get(ctx context.Context, id string) (*model.Playlist, error) {
	return s.ds.Playlist(ctx).Get(id)
}

func (s *playlists) GetWithTracks(ctx context.Context, id string) (*model.Playlist, error) {
	return s.ds.Playlist(ctx).GetWithTracks(id, true, false)
}

func (s *playlists) GetPlaylists(ctx context.Context, mediaFileId string) (model.Playlists, error) {
	return s.ds.Playlist(ctx).GetPlaylists(mediaFileId)
}

// --- Mutation operations ---

// Create creates a new playlist (when name is provided) or replaces tracks on an existing
// playlist (when playlistId is provided). This matches the Subsonic createPlaylist semantics.
func (s *playlists) Create(ctx context.Context, playlistId string, name string, ids []string) (string, error) {
	usr, _ := request.UserFrom(ctx)
	err := s.ds.WithTxImmediate(func(tx model.DataStore) error {
		var pls *model.Playlist
		var err error

		if playlistId != "" {
			pls, err = tx.Playlist(ctx).Get(playlistId)
			if err != nil {
				return err
			}
			if !usr.IsAdmin && pls.OwnerID != usr.ID {
				return model.ErrNotAuthorized
			}
		} else {
			pls = &model.Playlist{Name: name}
			pls.OwnerID = usr.ID
		}
		pls.Tracks = nil
		pls.AddMediaFilesByID(ids)

		err = tx.Playlist(ctx).Put(pls)
		playlistId = pls.ID
		return err
	})
	return playlistId, err
}

func (s *playlists) Delete(ctx context.Context, id string) error {
	if _, err := s.checkWritable(ctx, id); err != nil {
		return err
	}
	return s.ds.Playlist(ctx).Delete(id)
}

// --- Permission helpers ---

// checkWritable fetches the playlist and verifies the current user can modify it.
func (s *playlists) checkWritable(ctx context.Context, id string) (*model.Playlist, error) {
	pls, err := s.ds.Playlist(ctx).Get(id)
	if err != nil {
		return nil, err
	}
	usr, _ := request.UserFrom(ctx)
	if !usr.IsAdmin && pls.OwnerID != usr.ID {
		return nil, model.ErrNotAuthorized
	}
	return pls, nil
}

// checkTracksEditable verifies the user can modify tracks (ownership + not smart playlist).
func (s *playlists) checkTracksEditable(ctx context.Context, playlistID string) (*model.Playlist, error) {
	pls, err := s.checkWritable(ctx, playlistID)
	if err != nil {
		return nil, err
	}
	if pls.IsSmartPlaylist() {
		return nil, model.ErrNotAuthorized
	}
	return pls, nil
}

// updateMetadata applies optional metadata changes to a playlist and persists it.
// Accepts a DataStore parameter so it can be used inside transactions.
// The caller is responsible for permission checks.
func (s *playlists) updateMetadata(ctx context.Context, ds model.DataStore, pls *model.Playlist, name *string, comment *string, public *bool) error {
	if name != nil {
		pls.Name = *name
	}
	if comment != nil {
		pls.Comment = *comment
	}
	if public != nil {
		pls.Public = *public
	}
	return ds.Playlist(ctx).Put(pls)
}

// savePlaylist creates a new playlist, assigning the owner from context.
func (s *playlists) savePlaylist(ctx context.Context, pls *model.Playlist) (string, error) {
	usr, _ := request.UserFrom(ctx)
	pls.OwnerID = usr.ID
	pls.ID = "" // Force new creation
	err := s.ds.Playlist(ctx).Put(pls)
	if err != nil {
		return "", err
	}
	return pls.ID, nil
}

// updatePlaylistEntity updates playlist metadata with permission checks.
// Used by the REST API wrapper.
func (s *playlists) updatePlaylistEntity(ctx context.Context, id string, entity *model.Playlist, cols ...string) error {
	current, err := s.checkWritable(ctx, id)
	if err != nil {
		switch {
		case errors.Is(err, model.ErrNotFound):
			return rest.ErrNotFound
		case errors.Is(err, model.ErrNotAuthorized):
			return rest.ErrPermissionDenied
		default:
			return err
		}
	}
	usr, _ := request.UserFrom(ctx)
	if !usr.IsAdmin && entity.OwnerID != "" && entity.OwnerID != current.OwnerID {
		return rest.ErrPermissionDenied
	}
	// Apply ownership change (admin only)
	if entity.OwnerID != "" {
		current.OwnerID = entity.OwnerID
	}
	return s.updateMetadata(ctx, s.ds, current, &entity.Name, &entity.Comment, &entity.Public)
}

// --- Track management operations ---

func (s *playlists) AddTracks(ctx context.Context, playlistID string, ids []string) (int, error) {
	if _, err := s.checkTracksEditable(ctx, playlistID); err != nil {
		return 0, err
	}
	return s.ds.Playlist(ctx).Tracks(playlistID, false).Add(ids)
}

func (s *playlists) AddAlbums(ctx context.Context, playlistID string, albumIds []string) (int, error) {
	if _, err := s.checkTracksEditable(ctx, playlistID); err != nil {
		return 0, err
	}
	return s.ds.Playlist(ctx).Tracks(playlistID, false).AddAlbums(albumIds)
}

func (s *playlists) AddArtists(ctx context.Context, playlistID string, artistIds []string) (int, error) {
	if _, err := s.checkTracksEditable(ctx, playlistID); err != nil {
		return 0, err
	}
	return s.ds.Playlist(ctx).Tracks(playlistID, false).AddArtists(artistIds)
}

func (s *playlists) AddDiscs(ctx context.Context, playlistID string, discs []model.DiscID) (int, error) {
	if _, err := s.checkTracksEditable(ctx, playlistID); err != nil {
		return 0, err
	}
	return s.ds.Playlist(ctx).Tracks(playlistID, false).AddDiscs(discs)
}

func (s *playlists) RemoveTracks(ctx context.Context, playlistID string, trackIds []string) error {
	if _, err := s.checkTracksEditable(ctx, playlistID); err != nil {
		return err
	}
	return s.ds.Playlist(ctx).Tracks(playlistID, false).Delete(trackIds...)
}

func (s *playlists) ReorderTrack(ctx context.Context, playlistID string, pos int, newPos int) error {
	if _, err := s.checkTracksEditable(ctx, playlistID); err != nil {
		return err
	}
	return s.ds.Playlist(ctx).Tracks(playlistID, false).Reorder(pos, newPos)
}

// --- REST adapter (follows Share/Library pattern) ---

func (s *playlists) NewRepository(ctx context.Context) rest.Repository {
	return &playlistRepositoryWrapper{
		ctx:                ctx,
		PlaylistRepository: s.ds.Playlist(ctx),
		service:            s,
	}
}

// playlistRepositoryWrapper wraps the playlist repository as a thin REST-to-service adapter.
// It satisfies rest.Repository through the embedded PlaylistRepository (via ResourceRepository),
// and rest.Persistable by delegating to service methods for all mutations.
type playlistRepositoryWrapper struct {
	model.PlaylistRepository
	ctx     context.Context
	service *playlists
}

func (r *playlistRepositoryWrapper) Save(entity any) (string, error) {
	return r.service.savePlaylist(r.ctx, entity.(*model.Playlist))
}

func (r *playlistRepositoryWrapper) Update(id string, entity any, cols ...string) error {
	return r.service.updatePlaylistEntity(r.ctx, id, entity.(*model.Playlist), cols...)
}

func (r *playlistRepositoryWrapper) Delete(id string) error {
	err := r.service.Delete(r.ctx, id)
	switch {
	case errors.Is(err, model.ErrNotFound):
		return rest.ErrNotFound
	case errors.Is(err, model.ErrNotAuthorized):
		return rest.ErrPermissionDenied
	default:
		return err
	}
}

func (s *playlists) TracksRepository(ctx context.Context, playlistId string, refreshSmartPlaylist bool) rest.Repository {
	repo := s.ds.Playlist(ctx)
	tracks := repo.Tracks(playlistId, refreshSmartPlaylist)
	if tracks == nil {
		return nil
	}
	return tracks.(rest.Repository)
}

type nspFile struct {
	criteria.Criteria
	Name    string `json:"name"`
	Comment string `json:"comment"`
	Public  *bool  `json:"public"`
}

func (i *nspFile) UnmarshalJSON(data []byte) error {
	m := map[string]any{}
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
