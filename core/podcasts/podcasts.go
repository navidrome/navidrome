package podcasts

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	. "github.com/Masterminds/squirrel"
	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/server/events"
)

type Podcasts interface {
	CreateChannel(ctx context.Context, url string) (*model.PodcastChannel, error)
	DeleteChannel(ctx context.Context, id string) error
	RefreshChannel(ctx context.Context, id string) error
	RefreshAll(ctx context.Context) error
	SearchFeeds(ctx context.Context, query string) ([]FeedSearchResult, error)
	TopFeeds(ctx context.Context, country string) ([]FeedSearchResult, error)
	DownloadEpisode(ctx context.Context, episodeID string) error
	DeleteEpisode(ctx context.Context, id string) error
	RunRetention(ctx context.Context) error
}

type podcasts struct {
	ds     model.DataStore
	broker events.Broker
}

func New(ds model.DataStore, broker events.Broker) Podcasts {
	return &podcasts{ds: ds, broker: broker}
}

func (p *podcasts) CreateChannel(ctx context.Context, url string) (*model.PodcastChannel, error) {
	if url == "" {
		return nil, errors.New("feed url is required")
	}
	existing, err := p.ds.PodcastChannel(ctx).FindByUrl(url)
	if err == nil {
		return existing, nil
	}
	if !errors.Is(err, model.ErrNotFound) {
		return nil, err
	}

	channel := &model.PodcastChannel{
		Url:            url,
		Title:          url,
		Status:         model.PodcastChannelStatusNew,
		DownloadPolicy: model.PodcastDownloadPolicy(defaultDownloadPolicy()),
	}
	if err := p.ds.PodcastChannel(ctx).Put(channel); err != nil {
		return nil, fmt.Errorf("saving podcast channel: %w", err)
	}

	if err := p.RefreshChannel(ctx, channel.ID); err != nil {
		log.Warn(ctx, "Error doing initial refresh of podcast channel", "url", url, "id", channel.ID, err)
	}

	refreshed, err := p.ds.PodcastChannel(ctx).Get(channel.ID)
	if err != nil {
		log.Warn(ctx, "Error re-fetching podcast channel after initial refresh", "id", channel.ID, err)
		return channel, nil //nolint:nilerr
	}
	return refreshed, nil
}

func (p *podcasts) DeleteChannel(ctx context.Context, id string) error {
	// Look up episodes before deleting the channel, since that DB-cascades
	// their rows away - each needs to be unlinked from any playlist first,
	// or the playlist would be left with a dangling entry.
	episodes, err := p.ds.PodcastEpisode(ctx).GetAll(model.QueryOptions{Filters: Eq{"channel_id": id}})
	if err != nil {
		log.Warn(ctx, "Error listing episodes for deleted podcast channel", "id", id, err)
	}

	if err := p.ds.PodcastChannel(ctx).Delete(id); err != nil {
		return err
	}
	channelDir := filepath.Join(conf.Server.Podcasts.StorageFolder.String(), id)
	if err := os.RemoveAll(channelDir); err != nil {
		log.Warn(ctx, "Error removing downloaded episodes for deleted podcast channel", "id", id, "dir", channelDir, err)
	}
	for _, ep := range episodes {
		if err := p.ds.Playlist(ctx).RemoveItemFromPlaylists(ep.ID); err != nil {
			log.Warn(ctx, "Error removing deleted podcast episode from playlists", "id", ep.ID, err)
		}
	}
	p.notifyRefresh(ctx, "podcastChannel", id)
	p.notifyRefresh(ctx, "podcastEpisode")
	return nil
}

func (p *podcasts) SearchFeeds(ctx context.Context, query string) ([]FeedSearchResult, error) {
	if query == "" {
		return nil, errors.New("search query is required")
	}
	return searchFeeds(ctx, query)
}

func (p *podcasts) TopFeeds(ctx context.Context, country string) ([]FeedSearchResult, error) {
	return topFeeds(ctx, country)
}

func (p *podcasts) notifyRefresh(ctx context.Context, resource string, ids ...string) {
	if p.broker == nil {
		return
	}
	p.broker.SendBroadcastMessage(ctx, (&events.RefreshResource{}).With(resource, ids...))
}

func (p *podcasts) notifyDownload(ctx context.Context, ep *model.PodcastEpisode) {
	p.notifyRefresh(ctx, "podcastEpisode", ep.ID)
	if p.broker == nil {
		return
	}
	p.broker.SendBroadcastMessage(ctx, &events.PodcastDownloadStatus{
		EpisodeID: ep.ID,
		ChannelID: ep.ChannelID,
		Status:    string(ep.DownloadStatus),
		Error:     ep.ErrorMessage,
	})
}

func defaultDownloadPolicy() string {
	if conf.Server.Podcasts.DefaultDownloadPolicy == "" {
		return string(model.PodcastDownloadPolicyNone)
	}
	return conf.Server.Podcasts.DefaultDownloadPolicy
}
