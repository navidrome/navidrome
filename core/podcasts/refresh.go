package podcasts

import (
	"context"
	"errors"
	"time"

	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/server/events"
)

// RefreshChannel fetches the feed and writes the shared channel/episode rows - infrastructure any
// subscriber can trigger (not just admins), so every read/write in here runs against an admin
// context regardless of who called it, matching Subscribe/Unsubscribe's use of adminContext for
// the same shared-infrastructure reason.
func (p *podcasts) RefreshChannel(ctx context.Context, id string) error {
	ctx = adminContext(ctx)
	channel, err := p.ds.PodcastChannel(ctx).Get(id)
	if err != nil {
		return err
	}

	channel.Status = model.PodcastChannelStatusDownloading
	channel.ErrorMessage = ""
	if err := p.ds.PodcastChannel(ctx).Put(channel, "Status", "ErrorMessage"); err != nil {
		log.Error(ctx, "Error updating podcast channel status", "id", id, err)
	}
	p.notifyChannelRefreshStatus(ctx, channel, "refreshing")

	feed, err := fetchFeed(ctx, channel.Url)
	if err != nil {
		channel.Status = model.PodcastChannelStatusError
		channel.ErrorMessage = err.Error()
		_ = p.ds.PodcastChannel(ctx).Put(channel, "Status", "ErrorMessage")
		p.notifyChannelRefreshStatus(ctx, channel, "error")
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

	var newEpisodes []model.PodcastEpisode
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
		newEpisodes = append(newEpisodes, episode)
	}

	now := time.Now()
	channel.Status = model.PodcastChannelStatusCompleted
	channel.LastCheckedAt = &now
	if err := p.ds.PodcastChannel(ctx).Put(channel); err != nil {
		log.Error(ctx, "Error saving refreshed podcast channel", "id", id, err)
	}
	p.notifyChannelRefreshStatus(ctx, channel, "completed")
	if len(newEpisodes) > 0 {
		p.notifyRefresh(ctx, "podcastEpisode")
	}

	p.fanOutDownloads(ctx, *channel, newEpisodes)

	return nil
}

// fanOutDownloads applies each subscriber's own download policy to the episodes newly discovered
// by this refresh - unlike the old channel-level DownloadPolicy, different subscribers to the same
// shared channel can decide independently whether (and which of) these episodes get fetched.
// Called with ctx already carrying admin privileges (see RefreshChannel).
func (p *podcasts) fanOutDownloads(ctx context.Context, channel model.PodcastChannel, newEpisodes []model.PodcastEpisode) {
	if len(newEpisodes) == 0 {
		return
	}
	subs, err := p.ds.PodcastSubscription(ctx).FindByChannel(channel.ID)
	if err != nil {
		log.Error(ctx, "Error listing subscriptions to fan out new podcast episodes", "channel", channel.ID, err)
		return
	}
	for _, sub := range subs {
		p.downloadForSubscriber(ctx, channel, sub, newEpisodes)
	}
}

func (p *podcasts) notifyChannelRefreshStatus(ctx context.Context, channel *model.PodcastChannel, status string) {
	p.notifyRefresh(ctx, "podcastChannel", channel.ID)
	if p.broker == nil {
		return
	}
	p.broker.SendBroadcastMessage(ctx, &events.PodcastRefreshStatus{
		ChannelID: channel.ID,
		Status:    status,
		Error:     channel.ErrorMessage,
	})
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
