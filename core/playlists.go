package core

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/criteria"
	"github.com/navidrome/navidrome/model/request"
)

type Playlists interface {
	ImportFile(ctx context.Context, dir string, fname string) (*model.Playlist, error)
	Update(ctx context.Context, playlistID string, name *string, comment *string, public *bool, idsToAdd []string, idxToRemove []int) error
	ImportM3U(ctx context.Context, reader io.Reader) (*model.Playlist, error)
}

type playlists struct {
	ds model.DataStore
}

func NewPlaylists(ds model.DataStore) Playlists {
	return &playlists{ds: ds}
}

func (s *playlists) ImportFile(ctx context.Context, dir string, fname string) (*model.Playlist, error) {
	pls, err := s.parsePlaylist(ctx, fname, dir)
	if err != nil {
		log.Error(ctx, "Error parsing playlist", "path", filepath.Join(dir, fname), err)
		return nil, err
	}
	log.Debug("Found playlist", "name", pls.Name, "lastUpdated", pls.UpdatedAt, "path", pls.Path, "numTracks", len(pls.Tracks))
	err = s.updatePlaylist(ctx, pls)
	if err != nil {
		log.Error(ctx, "Error updating playlist", "path", filepath.Join(dir, fname), err)
	}
	return pls, err
}

func (s *playlists) ImportM3U(ctx context.Context, reader io.Reader) (*model.Playlist, error) {
	owner, _ := request.UserFrom(ctx)
	pls := &model.Playlist{
		OwnerID: owner.ID,
		Public:  false,
		Sync:    true,
	}
	pls, err := s.parseM3U(ctx, pls, "", reader)
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

func (s *playlists) parsePlaylist(ctx context.Context, playlistFile string, baseDir string) (*model.Playlist, error) {
	pls, err := s.newSyncedPlaylist(baseDir, playlistFile)
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
		return s.parseNSP(ctx, pls, file)
	default:
		return s.parseM3U(ctx, pls, baseDir, file)
	}
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

func (s *playlists) parseNSP(ctx context.Context, pls *model.Playlist, file io.Reader) (*model.Playlist, error) {
	nsp := &nspFile{}
	dec := json.NewDecoder(file)
	err := dec.Decode(nsp)
	if err != nil {
		log.Error(ctx, "Error parsing SmartPlaylist", "playlist", pls.Name, err)
		return nil, err
	}
	pls.Rules = &nsp.Criteria
	if nsp.Name != "" {
		pls.Name = nsp.Name
	}
	if nsp.Comment != "" {
		pls.Comment = nsp.Comment
	}
	return pls, nil
}

func (s *playlists) parseM3U(ctx context.Context, pls *model.Playlist, baseDir string, reader io.Reader) (*model.Playlist, error) {
	mediaFileRepository := s.ds.MediaFile(ctx)
	scanner := bufio.NewScanner(reader)
	scanner.Split(scanLines)
	var mfs model.MediaFiles
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "#PLAYLIST:") {
			if split := strings.Split(line, ":"); len(split) >= 2 {
				pls.Name = split[1]
			}
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
		if baseDir != "" && !filepath.IsAbs(line) {
			line = filepath.Join(baseDir, line)
		}
		mf, err := mediaFileRepository.FindByPath(line)
		if err != nil {
			log.Warn(ctx, "Path in playlist not found", "playlist", pls.Name, "path", line, err)
			continue
		}
		mfs = append(mfs, *mf)
	}
	if pls.Name == "" {
		pls.Name = time.Now().Format(time.RFC3339)
	}
	pls.Tracks = nil
	pls.AddMediaFiles(mfs)

	return pls, scanner.Err()
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
	}
	return s.ds.Playlist(ctx).Put(newPls)
}

// From https://stackoverflow.com/a/41433698
func scanLines(data []byte, atEOF bool) (advance int, token []byte, err error) {
	if atEOF && len(data) == 0 {
		return 0, nil, nil
	}
	if i := bytes.IndexAny(data, "\r\n"); i >= 0 {
		if data[i] == '\n' {
			// We have a line terminated by single newline.
			return i + 1, data[0:i], nil
		}
		advance = i + 1
		if len(data) > i+1 && data[i+1] == '\n' {
			advance += 1
		}
		return advance, data[0:i], nil
	}
	// If we're at EOF, we have a final, non-terminated line. Return it.
	if atEOF {
		return len(data), data, nil
	}
	// Request more data.
	return 0, nil, nil
}

func (s *playlists) Update(ctx context.Context, playlistID string,
	name *string, comment *string, public *bool,
	idsToAdd []string, idxToRemove []int) error {
	needsInfoUpdate := name != nil || comment != nil || public != nil
	needsTrackRefresh := len(idxToRemove) > 0

	return s.ds.WithTx(func(tx model.DataStore) error {
		var pls *model.Playlist
		var err error
		repo := tx.Playlist(ctx)
		if needsTrackRefresh {
			pls, err = repo.GetWithTracks(playlistID, true)
			pls.RemoveTracks(idxToRemove)
			pls.AddTracks(idsToAdd)
		} else {
			if len(idsToAdd) > 0 {
				_, err = repo.Tracks(playlistID, true).Add(idsToAdd)
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
			if err = repo.Tracks(playlistID, true).DeleteAll(); err != nil {
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
