package filesystem

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Masterminds/squirrel"
	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/core/agents"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/utils"
)

const filesystemAgentName = "filesystem"

var (
	supportedExtensions = []string{"lrc", "txt"}
)

type filesystemAgent struct {
	ds model.DataStore
}

func filesystemConstructor(ds model.DataStore) *filesystemAgent {
	return &filesystemAgent{
		ds: ds,
	}
}

func (f *filesystemAgent) AgentName() string {
	return filesystemAgentName
}

func (f *filesystemAgent) GetArtistBiography(ctx context.Context, id, name, mbid string) (string, error) {
	_, err := f.ds.Artist(ctx).Get(id)
	if err != nil {
		return "", err
	}
	als, err := f.ds.Album(ctx).GetAll(model.QueryOptions{Filters: squirrel.Eq{"album_artist_id": id}})
	if err != nil {
		return "", err
	}
	var paths []string
	for _, al := range als {
		paths = append(paths, strings.Split(al.Paths, consts.Zwsp)...)
	}
	artistFolder := utils.LongestCommonPrefix(paths)
	if !strings.HasSuffix(artistFolder, string(filepath.Separator)) {
		artistFolder, _ = filepath.Split(artistFolder)
	}
	artistBioPath := filepath.Join(artistFolder, "artist.txt")
	contents, err := os.ReadFile(artistBioPath)
	if err != nil {
		return "", agents.ErrNotFound
	}
	return string(contents), nil
}

func (f *filesystemAgent) GetSongLyrics(ctx context.Context, mf *model.MediaFile) (model.LyricList, error) {
	lyrics := model.LyricList{}
	extension := filepath.Ext(mf.Path)
	basePath := mf.Path[0 : len(mf.Path)-len(extension)]

	for _, ext := range supportedExtensions {
		lrcPath := fmt.Sprintf("%s.%s", basePath, ext)
		contents, err := os.ReadFile(lrcPath)

		if err != nil {
			continue
		}

		lyric, err := model.ToLyrics("xxx", string(contents))
		if err != nil {
			return nil, err
		}

		lyrics = append(lyrics, *lyric)
	}

	return lyrics, nil
}

func init() {
	conf.AddHook(func() {
		agents.Register(filesystemAgentName, func(ds model.DataStore) agents.Interface {
			return filesystemConstructor(ds)
		})
	})
}

var _ agents.ArtistBiographyRetriever = (*filesystemAgent)(nil)
var _ agents.LyricsRetriever = (*filesystemAgent)(nil)
