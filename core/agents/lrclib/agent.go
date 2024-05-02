package lrclib

import (
	"context"
	"errors"
	"net/http"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/core/agents"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/utils"
)

const (
	lrclibAgentName = "lrclib"
)

type lrclibAgent struct {
	client *client
}

func (l *lrclibAgent) GetSongLyrics(ctx context.Context, mf *model.MediaFile) (model.LyricList, error) {
	lyrics, err := l.client.getLyrics(ctx, mf.Title, mf.AlbumArtist, mf.Album, mf.Duration)

	if err != nil {
		var lrclibError *lrclibError
		isLrclibError := errors.As(err, &lrclibError)

		if isLrclibError && lrclibError.Code == 404 {
			log.Info(ctx, "Track not present in LrcLib")
			return nil, agents.ErrNotFound
		}

		log.Error(ctx, "Error fetching lyrics", "id", mf.ID, err)
		return nil, err
	}

	songLyrics := model.LyricList{}

	if lyrics.Instrumental {
		return nil, nil
	}

	if lyrics.SyncedLyrics != "" {
		lyrics, err := model.ToLyrics("xxx", lyrics.SyncedLyrics)
		if err != nil {
			return nil, err
		}

		songLyrics = append(songLyrics, *lyrics)
	}

	if lyrics.PlainLyrics != "" {
		lyrics, err := model.ToLyrics("xxx", lyrics.PlainLyrics)
		if err != nil {
			return nil, err
		}

		songLyrics = append(songLyrics, *lyrics)
	}

	return songLyrics, nil
}

func lrclibConstructor() *lrclibAgent {
	l := &lrclibAgent{}
	hc := &http.Client{
		Timeout: consts.DefaultHttpClientTimeOut,
	}
	chc := utils.NewCachedHTTPClient(hc, consts.DefaultHttpClientTimeOut)
	l.client = newClient(chc)
	return l
}

func (l *lrclibAgent) AgentName() string {
	return lrclibAgentName
}

func init() {
	conf.AddHook(func() {
		agents.Register(lrclibAgentName, func(ds model.DataStore) agents.Interface {
			return lrclibConstructor()
		})
	})
}

var _ agents.LyricsRetriever = (lrclibConstructor)()
