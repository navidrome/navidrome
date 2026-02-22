package playlists

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/RaveNoX/go-jsoncommentstrip"
	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/criteria"
)

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
