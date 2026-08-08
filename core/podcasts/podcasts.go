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
	"github.com/navidrome/navidrome/model/request"
	"github.com/navidrome/navidrome/server/events"
)

type Podcasts interface {
	// Subscribe ensures the shared channel for url exists (creating it if this is the first
	// subscriber to this feed), then creates a subscription for the current user (from ctx) with
	// their configured default download policy. Idempotent - subscribing again just returns the
	// existing subscription.
	Subscribe(ctx context.Context, url string) (*model.PodcastChannel, error)
	// Unsubscribe removes the current user's (from ctx) subscription to the given channel. If
	// they were the last remaining subscriber, the shared channel and its downloaded files are
	// torn down too - nobody else has any claim on them anymore.
	Unsubscribe(ctx context.Context, channelID string) error
	// UpdateSubscription saves changes to one of the current user's own subscriptions (download
	// policy, retention limits) - ownership is verified against ctx's user, not trusted from sub.
	UpdateSubscription(ctx context.Context, sub *model.PodcastSubscription) error
	RefreshChannel(ctx context.Context, id string) error
	RefreshAll(ctx context.Context) error
	SearchFeeds(ctx context.Context, query string) ([]FeedSearchResult, error)
	TopFeeds(ctx context.Context, country string) ([]FeedSearchResult, error)
	// DownloadEpisode marks the episode as downloaded for the current user (from ctx). If the
	// shared file doesn't already exist on disk, this is also what triggers fetching it - if it
	// does, this is the entire operation (no re-fetch), which is what lets a second subscriber's
	// "download" click resolve instantly from another subscriber's earlier download.
	DownloadEpisode(ctx context.Context, episodeID string) error
	// DeleteEpisode clears the current user's (from ctx) own downloaded flag for the episode. The
	// shared file itself is only actually deleted once no subscriber anywhere still has it
	// flagged - see retention.go's cleanupOrphanedFiles.
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

// adminContext returns a context with admin privileges, for DB operations that must proceed
// regardless of which (possibly non-admin) user's request triggered them - e.g. an ordinary user
// unsubscribing as the last subscriber to a channel, which needs to delete shared infrastructure
// the persistence layer otherwise gates to admins. Mirrors plugins.adminContext.
func adminContext(ctx context.Context) context.Context {
	return request.WithUser(ctx, model.User{IsAdmin: true})
}

func (p *podcasts) Subscribe(ctx context.Context, url string) (*model.PodcastChannel, error) {
	if url == "" {
		return nil, errors.New("feed url is required")
	}
	user, ok := request.UserFrom(ctx)
	if !ok {
		return nil, errors.New("subscribing requires an authenticated user")
	}

	// Finding/creating the shared channel is infrastructure, not a personal action - runs against
	// an admin context so it works regardless of whether the caller is an admin, and so a
	// non-admin caller's own scoped FindByUrl (which would otherwise only ever see channels they
	// already subscribe to, since selectChannel inner-joins subscriptions for regular users)
	// doesn't wrongly conclude an already-shared channel doesn't exist yet and create a duplicate.
	channel, err := p.ds.PodcastChannel(adminContext(ctx)).FindByUrl(url)
	isNewChannel := false
	switch {
	case errors.Is(err, model.ErrNotFound):
		isNewChannel = true
		channel = &model.PodcastChannel{
			Url:    url,
			Title:  url,
			Status: model.PodcastChannelStatusNew,
		}
		if err := p.ds.PodcastChannel(adminContext(ctx)).Put(channel); err != nil {
			return nil, fmt.Errorf("saving podcast channel: %w", err)
		}
	case err != nil:
		return nil, err
	}

	if _, err := p.ds.PodcastSubscription(ctx).FindByChannelAndUser(channel.ID, user.ID); err == nil {
		// Already subscribed - idempotent. Re-fetch on the caller's own ctx (not the adminContext
		// used above) so the returned channel's Subscription field reflects the caller's own
		// subscription rather than coming back empty.
		if own, err := p.ds.PodcastChannel(ctx).Get(channel.ID); err == nil {
			return own, nil
		}
		return channel, nil
	} else if !errors.Is(err, model.ErrNotFound) {
		return nil, err
	}

	sub := &model.PodcastSubscription{
		ChannelID:      channel.ID,
		UserID:         user.ID,
		DownloadPolicy: model.PodcastDownloadPolicy(defaultDownloadPolicy()),
	}
	if err := p.ds.PodcastSubscription(ctx).Put(sub); err != nil {
		return nil, fmt.Errorf("saving podcast subscription: %w", err)
	}

	if isNewChannel {
		if err := p.RefreshChannel(ctx, channel.ID); err != nil {
			log.Warn(ctx, "Error doing initial refresh of podcast channel", "url", url, "id", channel.ID, err)
		}
	} else if sub.DownloadPolicy == model.PodcastDownloadPolicyAll {
		// Existing channel, new subscriber, backfill policy: download the full existing episode
		// backlog for them. Deliberately not done for policy New - "new" means "starting from
		// now on," not "backfill everything that already exists" - only All backfills.
		existingEpisodes, err := p.ds.PodcastEpisode(adminContext(ctx)).GetAll(model.QueryOptions{Filters: Eq{"channel_id": channel.ID}})
		if err != nil {
			log.Warn(ctx, "Error listing episodes for new subscriber's backfill", "channelId", channel.ID, err)
		} else {
			p.downloadForSubscriber(ctx, *channel, *sub, existingEpisodes)
		}
	}

	p.notifyRefresh(ctx, "podcastChannel", channel.ID)

	refreshed, err := p.ds.PodcastChannel(ctx).Get(channel.ID)
	if err != nil {
		log.Warn(ctx, "Error re-fetching podcast channel after subscribe", "id", channel.ID, err)
		return channel, nil //nolint:nilerr
	}
	return refreshed, nil
}

func (p *podcasts) Unsubscribe(ctx context.Context, channelID string) error {
	user, ok := request.UserFrom(ctx)
	if !ok {
		return errors.New("unsubscribing requires an authenticated user")
	}

	sub, err := p.ds.PodcastSubscription(ctx).FindByChannelAndUser(channelID, user.ID)
	if err != nil {
		return err
	}

	remaining, err := p.ds.PodcastSubscription(ctx).Delete(sub.ID)
	if err != nil {
		return err
	}
	p.notifyRefresh(ctx, "podcastChannel", channelID)

	if remaining > 0 {
		return nil
	}

	// Last subscriber left - tear down the shared channel and its files. Runs against an
	// admin-context repository instance rather than the unsubscribing user's own ctx, since
	// deleting shared infrastructure is admin-gated at the persistence layer and an ordinary
	// user's unsubscribe shouldn't need admin rights for the cleanup that naturally follows it.
	return p.deleteOrphanedChannel(adminContext(ctx), channelID)
}

// deleteOrphanedChannel removes a channel (and, via cascade, its episodes) once confirmed to have
// zero remaining subscriptions - the direct counterpart to the old admin-only DeleteChannel,
// called automatically by Unsubscribe rather than exposed as its own API surface. ctx must
// already carry admin/system privileges (see adminContext).
func (p *podcasts) deleteOrphanedChannel(ctx context.Context, id string) error {
	episodes, err := p.ds.PodcastEpisode(ctx).GetAll(model.QueryOptions{Filters: Eq{"channel_id": id}})
	if err != nil {
		log.Warn(ctx, "Error listing episodes for orphaned podcast channel", "id", id, err)
	}

	if err := p.ds.PodcastChannel(ctx).Delete(id); err != nil {
		return err
	}
	channelDir := filepath.Join(conf.Server.Podcasts.StorageFolder.String(), id)
	if err := os.RemoveAll(channelDir); err != nil {
		log.Warn(ctx, "Error removing downloaded episodes for orphaned podcast channel", "id", id, "dir", channelDir, err)
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

func (p *podcasts) UpdateSubscription(ctx context.Context, sub *model.PodcastSubscription) error {
	user, ok := request.UserFrom(ctx)
	if !ok {
		return errors.New("updating a subscription requires an authenticated user")
	}
	existing, err := p.ds.PodcastSubscription(ctx).Get(sub.ID)
	if err != nil {
		return err
	}
	if existing.UserID != user.ID && !user.IsAdmin {
		return model.ErrNotAuthorized
	}
	sub.ID = existing.ID
	sub.ChannelID = existing.ChannelID
	sub.UserID = existing.UserID
	if err := p.ds.PodcastSubscription(ctx).Put(sub); err != nil {
		return err
	}
	p.notifyRefresh(ctx, "podcastChannel", existing.ChannelID)
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
