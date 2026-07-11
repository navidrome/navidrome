package podcasts

import (
	"context"
	"errors"
	"time"

	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
)

func (p *podcasts) RefreshChannel(ctx context.Context, id string) error {
	channel, err := p.ds.PodcastChannel(ctx).Get(id)
	if err != nil {
		return err
	}

	channel.Status = model.PodcastChannelStatusDownloading
	channel.ErrorMessage = ""
	if err := p.ds.PodcastChannel(ctx).Put(channel, "Status", "ErrorMessage"); err != nil {
		log.Error(ctx, "Error updating podcast channel status", "id", id, err)
	}
	p.notifyRefresh(ctx, "podcastChannel", id)

	feed, err := fetchFeed(ctx, channel.Url)
	if err != nil {
		channel.Status = model.PodcastChannelStatusError
		channel.ErrorMessage = err.Error()
		_ = p.ds.PodcastChannel(ctx).Put(channel, "Status", "ErrorMessage")
		p.notifyRefresh(ctx, "podcastChannel", id)
		return err
	}

	title, description, homePage, imageUrl := feedChannelInfo(feed)
	if title != "" {
		channel.Title = title
	}
	channel.Description = description
	channel.HomePageUrl = homePage
	channel.CoverArtUrl = imageUrl
	channel.OriginalImageUrl = imageUrl

	var newEpisodeIDs []string
	for _, item := range feed.Items {
		episode := feedItemToEpisode(id, item)
		existing, err := p.ds.PodcastEpisode(ctx).FindByGuid(id, episode.Guid)
		if err == nil {
			episode.ID = existing.ID
			episode.DownloadStatus = existing.DownloadStatus
			episode.Path = existing.Path
			episode.Suffix = existing.Suffix
			episode.Size = existing.Size
			episode.BitRate = existing.BitRate
			if err := p.ds.PodcastEpisode(ctx).Put(&episode); err != nil {
				log.Error(ctx, "Error updating podcast episode", "channel", id, "guid", episode.Guid, err)
			}
			continue
		}
		if !errors.Is(err, model.ErrNotFound) {
			log.Error(ctx, "Error looking up podcast episode", "channel", id, "guid", episode.Guid, err)
			continue
		}
		if err := p.ds.PodcastEpisode(ctx).Put(&episode); err != nil {
			log.Error(ctx, "Error saving new podcast episode", "channel", id, "guid", episode.Guid, err)
			continue
		}
		newEpisodeIDs = append(newEpisodeIDs, episode.ID)
	}

	now := time.Now()
	channel.Status = model.PodcastChannelStatusCompleted
	channel.LastCheckedAt = &now
	if err := p.ds.PodcastChannel(ctx).Put(channel); err != nil {
		log.Error(ctx, "Error saving refreshed podcast channel", "id", id, err)
	}
	p.notifyRefresh(ctx, "podcastChannel", id)
	if len(newEpisodeIDs) > 0 {
		p.notifyRefresh(ctx, "podcastEpisode")
	}

	// Downloading new/backfilled episodes to disk is implemented in a later
	// phase; Phase 1 only keeps the episode metadata in sync with the feed.
	_ = newEpisodeIDs

	return nil
}

func (p *podcasts) RefreshAll(ctx context.Context) error {
	channels, err := p.ds.PodcastChannel(ctx).GetAll()
	if err != nil {
		return err
	}
	var firstErr error
	for _, channel := range channels {
		if err := p.RefreshChannel(ctx, channel.ID); err != nil {
			log.Error(ctx, "Error refreshing podcast channel", "id", channel.ID, "url", channel.Url, err)
			if firstErr == nil {
				firstErr = err
			}
		}
	}
	return firstErr
}
