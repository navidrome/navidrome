package listenbrainz

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/core/agents"
	"github.com/navidrome/navidrome/core/external_playlists"
	"github.com/navidrome/navidrome/core/scrobbler"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/utils"
)

const (
	listenBrainzAgentName = "listenbrainz"
	sessionKeyProperty    = "ListenBrainzSessionKey"
	troiBot               = "troi-bot"
	playlistTypeUser      = "user"
	playlistTypeCollab    = "collab"
	playlistTypeCreated   = "created"
	defaultFetch          = 25
	sourceDaily           = "daily-jams"
)

var (
	playlistTypes = []string{playlistTypeUser, playlistTypeCollab, playlistTypeCreated}
)

type listenBrainzAgent struct {
	ds          model.DataStore
	sessionKeys *agents.SessionKeys
	baseURL     string
	client      *client
}

func listenBrainzConstructor(ds model.DataStore) *listenBrainzAgent {
	l := &listenBrainzAgent{
		ds:          ds,
		sessionKeys: &agents.SessionKeys{DataStore: ds, KeyName: sessionKeyProperty},
		baseURL:     conf.Server.ListenBrainz.BaseURL,
	}
	hc := &http.Client{
		Timeout: consts.DefaultHttpClientTimeOut,
	}
	chc := utils.NewCachedHTTPClient(hc, consts.DefaultHttpClientTimeOut)
	l.client = newClient(l.baseURL, chc)
	return l
}

func (l *listenBrainzAgent) AgentName() string {
	return listenBrainzAgentName
}

func (l *listenBrainzAgent) formatListen(track *model.MediaFile) listenInfo {
	li := listenInfo{
		TrackMetadata: trackMetadata{
			ArtistName:  track.Artist,
			TrackName:   track.Title,
			ReleaseName: track.Album,
			AdditionalInfo: additionalInfo{
				SubmissionClient:        consts.AppName,
				SubmissionClientVersion: consts.Version,
				TrackNumber:             track.TrackNumber,
				ArtistMbzIDs:            []string{track.MbzArtistID},
				TrackMbzID:              track.MbzTrackID,
				ReleaseMbID:             track.MbzAlbumID,
			},
		},
	}
	return li
}

func (l *listenBrainzAgent) NowPlaying(ctx context.Context, userId string, track *model.MediaFile) error {
	sk, err := l.sessionKeys.Get(ctx, userId)
	if err != nil || sk == "" {
		return scrobbler.ErrNotAuthorized
	}

	li := l.formatListen(track)
	err = l.client.updateNowPlaying(ctx, sk, li)
	if err != nil {
		log.Warn(ctx, "ListenBrainz updateNowPlaying returned error", "track", track.Title, err)
		return scrobbler.ErrUnrecoverable
	}
	return nil
}

func (l *listenBrainzAgent) Scrobble(ctx context.Context, userId string, s scrobbler.Scrobble) error {
	sk, err := l.sessionKeys.Get(ctx, userId)
	if err != nil || sk == "" {
		return scrobbler.ErrNotAuthorized
	}

	li := l.formatListen(&s.MediaFile)
	li.ListenedAt = int(s.TimeStamp.Unix())
	err = l.client.scrobble(ctx, sk, li)

	if err == nil {
		return nil
	}
	var lbErr *listenBrainzError
	isListenBrainzError := errors.As(err, &lbErr)
	if !isListenBrainzError {
		log.Warn(ctx, "ListenBrainz Scrobble returned HTTP error", "track", s.Title, err)
		return scrobbler.ErrRetryLater
	}
	if lbErr.Code == 500 || lbErr.Code == 503 {
		return scrobbler.ErrRetryLater
	}
	return scrobbler.ErrUnrecoverable
}

func (l *listenBrainzAgent) GetPlaylistTypes() []string {
	return playlistTypes
}

func getIdentifier(url string) string {
	split := strings.Split(url, "/")
	return split[len(split)-1]
}

func (l *listenBrainzAgent) GetPlaylists(ctx context.Context, offset, count int, userId, playlistType string) (*external_playlists.ExternalPlaylists, error) {
	token, err := l.sessionKeys.GetWithUser(ctx, userId)

	if errors.Is(agents.ErrNoUsername, err) {
		resp, err := l.client.validateToken(ctx, token.Key)

		if err != nil {
			return nil, err
		}

		token.User = resp.UserName

		err = l.sessionKeys.PutWithUser(ctx, userId, token.Key, resp.UserName)
		if err != nil {
			return nil, err
		}
	} else if err != nil {
		return nil, err
	}

	resp, err := l.client.getPlaylists(ctx, offset, count, token.Key, token.User, playlistType)

	if err != nil {
		return nil, err
	}

	lists := make([]external_playlists.ExternalPlaylist, len(resp.Playlists))

	for i, playlist := range resp.Playlists {
		pls := playlist.Playlist

		lists[i] = external_playlists.ExternalPlaylist{
			Name:        pls.Title,
			Description: utils.SanitizeText(pls.Annotation),
			Creator:     pls.Creator,
			ID:          getIdentifier(pls.Identifier),
			Url:         pls.Identifier,
			CreatedAt:   pls.Date,
			UpdatedAt:   pls.Extension.Extension.LastModified,
			Syncable:    pls.Creator != troiBot,
		}
	}

	return &external_playlists.ExternalPlaylists{
		Total: resp.PlaylistCount,
		Lists: lists,
	}, nil
}

func (l *listenBrainzAgent) ImportPlaylist(ctx context.Context, update bool, sync bool, userId, id, name string) error {
	token, err := l.sessionKeys.Get(ctx, userId)
	if err != nil {
		return err
	}

	pls, err := l.client.getPlaylist(ctx, token, id)
	if err != nil {
		return err
	}

	syncable := pls.Playlist.Creator != troiBot

	if sync && !syncable {
		return external_playlists.ErrSyncUnsupported
	}

	err = l.ds.WithTx(func(tx model.DataStore) error {
		ids := make([]string, len(pls.Playlist.Tracks))
		for i, track := range pls.Playlist.Tracks {
			ids[i] = getIdentifier(track.Identifier)
		}

		matched_tracks, err := tx.MediaFile(ctx).FindWithMbid(ids)

		if err != nil {
			return err
		}

		var playlist *model.Playlist = nil

		comment := agents.StripAllTags.Sanitize(pls.Playlist.Annotation)

		if update {
			playlist, err = tx.Playlist(ctx).GetByExternalInfo(listenBrainzAgentName, id)

			if err != nil {
				if !errors.Is(err, model.ErrNotFound) {
					log.Error(ctx, "Failed to query for playlist", "error", err)
				}
			} else if playlist.ExternalAgent != listenBrainzAgentName {
				return fmt.Errorf("existing agent %s does not match current agent %s", playlist.ExternalAgent, listenBrainzAgentName)
			} else if userId != playlist.OwnerID {
				return model.ErrNotAuthorized
			} else {
				playlist.Name = name
				playlist.Comment = comment
			}
		}

		if playlist == nil {
			playlist = &model.Playlist{
				Name:             name,
				Comment:          comment,
				OwnerID:          userId,
				Public:           false,
				ExternalAgent:    listenBrainzAgentName,
				ExternalId:       id,
				ExternalSync:     sync,
				ExternalSyncable: syncable,
				ExternalUrl:      pls.Playlist.Identifier,
			}
		}

		playlist.AddMediaFiles(matched_tracks)

		err = tx.Playlist(ctx).Put(playlist)

		if err != nil {
			log.Error(ctx, "Failed to import playlist", "id", id, err)
		}

		return err
	})

	return err
}

func (l *listenBrainzAgent) SyncPlaylist(ctx context.Context, tx model.DataStore, pls *model.Playlist) error {
	token, err := l.sessionKeys.Get(ctx, pls.OwnerID)
	if err != nil {
		return err
	}

	external, err := l.client.getPlaylist(ctx, token, pls.ExternalId)
	if err != nil {
		return err
	}

	ids := make([]string, len(external.Playlist.Tracks))
	for i, track := range external.Playlist.Tracks {
		ids[i] = getIdentifier(track.Identifier)
	}

	matched_tracks, err := tx.MediaFile(ctx).FindWithMbid(ids)

	if err != nil {
		return err
	}

	comment := agents.StripAllTags.Sanitize(external.Playlist.Annotation)

	pls.Comment = comment

	pls.AddMediaFiles(matched_tracks)

	err = tx.Playlist(ctx).Put(pls)

	if err != nil {
		log.Error(ctx, "Failed to sync playlist", "id", pls.ID, err)
	}

	return err
}

func (l *listenBrainzAgent) SyncRecommended(ctx context.Context, userId string) error {
	token, err := l.sessionKeys.GetWithUser(ctx, userId)

	if errors.Is(agents.ErrNoUsername, err) {
		resp, err := l.client.validateToken(ctx, token.Key)

		if err != nil {
			return err
		}

		token.User = resp.UserName

		err = l.sessionKeys.PutWithUser(ctx, userId, token.Key, resp.UserName)
		if err != nil {
			return err
		}
	} else if err != nil {
		return err
	}

	resp, err := l.client.getPlaylists(ctx, 0, defaultFetch, token.Key, token.User, playlistTypeCreated)

	if err != nil {
		return err
	}

	var full_pls *listenBrainzResponse = nil
	var id string

	for _, pls := range resp.Playlists {
		if pls.Playlist.Extension.Extension.AdditionalMetadata.AlgorithmMetadata.SourcePatch == sourceDaily {
			id = getIdentifier(pls.Playlist.Identifier)

			full_pls, err = l.client.getPlaylist(ctx, token.Key, id)
			break
		}
	}

	if err != nil {
		return err
	} else if full_pls == nil {
		return agents.ErrNotFound
	}

	err = l.ds.WithTx(func(tx model.DataStore) error {
		ids := make([]string, len(full_pls.Playlist.Tracks))
		for i, track := range full_pls.Playlist.Tracks {
			ids[i] = getIdentifier(track.Identifier)
		}

		matched_tracks, err := tx.MediaFile(ctx).FindWithMbid(ids)

		if err != nil {
			return err
		}

		playlist, err := tx.Playlist(ctx).GetRecommended(userId, listenBrainzAgentName)

		comment := agents.StripAllTags.Sanitize(full_pls.Playlist.Annotation)

		if err != nil {
			playlist = &model.Playlist{
				Name:                "ListenBrainz Daily Playlist",
				Comment:             comment,
				OwnerID:             userId,
				Public:              false,
				ExternalAgent:       listenBrainzAgentName,
				ExternalId:          id,
				ExternalSync:        false,
				ExternalSyncable:    false,
				ExternalRecommended: true,
			}

			if !errors.Is(err, model.ErrNotFound) {
				log.Error(ctx, "Failed to query for playlist", "error", err)
			}
		}

		playlist.ExternalId = id
		playlist.ExternalUrl = full_pls.Playlist.Identifier

		playlist.AddMediaFiles(matched_tracks)
		err = tx.Playlist(ctx).Put(playlist)

		if err != nil {
			log.Error(ctx, "Failed to import playlist", "id", id, err)
		}

		return err
	})

	return err
}

func (l *listenBrainzAgent) IsAuthorized(ctx context.Context, userId string) bool {
	sk, err := l.sessionKeys.Get(ctx, userId)
	return err == nil && sk != ""
}

func init() {
	conf.AddHook(func() {
		if conf.Server.ListenBrainz.Enabled {
			scrobbler.Register(listenBrainzAgentName, func(ds model.DataStore) scrobbler.Scrobbler {
				return listenBrainzConstructor(ds)
			})

			external_playlists.Register(listenBrainzAgentName, func(ds model.DataStore) external_playlists.PlaylistAgent {
				return listenBrainzConstructor(ds)
			})
		}
	})
}
