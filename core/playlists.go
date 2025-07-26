package core

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/RaveNoX/go-jsoncommentstrip"
	"github.com/bmatcuk/doublestar/v4"
	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/criteria"
	"github.com/navidrome/navidrome/model/request"
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

	extension := strings.ToLower(filepath.Ext(playlistFile))
	switch extension {
	case ".nsp":
		err = s.parseNSP(ctx, pls, file)
	default:
		err = s.parseM3U(ctx, pls, folder, file)
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
		paths, err := s.normalizePaths(ctx, pls, folder, filteredLines)
		if err != nil {
			log.Warn(ctx, "Error normalizing paths in playlist", "playlist", pls.Name, err)
			continue
		}
		found, err := mediaFileRepository.FindByPaths(paths)
		if err != nil {
			log.Warn(ctx, "Error reading files from DB", "playlist", pls.Name, err)
			continue
		}
		existing := make(map[string]int, len(found))
		for idx := range found {
			existing[normalizePathForComparison(found[idx].Path)] = idx
		}
		for _, path := range paths {
			idx, ok := existing[normalizePathForComparison(path)]
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

// normalizePathForComparison normalizes a file path to NFC form and converts to lowercase
// for consistent comparison. This fixes Unicode normalization issues on macOS where
// Apple Music creates playlists with NFC-encoded paths but the filesystem uses NFD.
func normalizePathForComparison(path string) string {
	return strings.ToLower(norm.NFC.String(path))
}

// TODO This won't work for multiple libraries
func (s *playlists) normalizePaths(ctx context.Context, pls *model.Playlist, folder *model.Folder, lines []string) ([]string, error) {
	libRegex, err := s.compileLibraryPaths(ctx)
	if err != nil {
		return nil, err
	}

	res := make([]string, 0, len(lines))
	for idx, line := range lines {
		var libPath string
		var filePath string

		if folder != nil && !filepath.IsAbs(line) {
			libPath = folder.LibraryPath
			filePath = filepath.Join(folder.AbsolutePath(), line)
		} else {
			cleanLine := filepath.Clean(line)
			if libPath = libRegex.FindString(cleanLine); libPath != "" {
				filePath = cleanLine
			}
		}

		if libPath != "" {
			if rel, err := filepath.Rel(libPath, filePath); err == nil {
				res = append(res, rel)
			} else {
				log.Debug(ctx, "Error getting relative path", "playlist", pls.Name, "path", line, "libPath", libPath,
					"filePath", filePath, err)
			}
		} else {
			log.Warn(ctx, "Path in playlist not found in any library", "path", line, "line", idx)
		}
	}
	return slice.Map(res, filepath.ToSlash), nil
}

func (s *playlists) compileLibraryPaths(ctx context.Context) (*regexp.Regexp, error) {
	libs, err := s.ds.Library(ctx).GetAll()
	if err != nil {
		return nil, err
	}

	// Create regex patterns for each library path
	patterns := make([]string, len(libs))
	for i, lib := range libs {
		cleanPath := filepath.Clean(lib.Path)
		escapedPath := regexp.QuoteMeta(cleanPath)
		patterns[i] = fmt.Sprintf("^%s(?:/|$)", escapedPath)
	}
	// Combine all patterns into a single regex
	combinedPattern := strings.Join(patterns, "|")
	re, err := regexp.Compile(combinedPattern)
	if err != nil {
		return nil, fmt.Errorf("compiling library paths `%s`: %w", combinedPattern, err)
	}
	return re, nil
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
		newPls.Public = conf.Server.DefaultPlaylistPublicVisibility
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
}

func (i *nspFile) UnmarshalJSON(data []byte) error {
	m := map[string]interface{}{}
	err := json.Unmarshal(data, &m)
	if err != nil {
		return err
	}
	i.Name, _ = m["name"].(string)
	i.Comment, _ = m["comment"].(string)
	return json.Unmarshal(data, &i.Criteria)
}
