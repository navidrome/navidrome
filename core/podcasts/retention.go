package podcasts

import (
	"context"
	"time"

	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
)

// RunRetention enforces each subscription's own RetentionCount, RetentionDays and MaxStorageMB
// policies (0 = unlimited) against that subscriber's own downloaded episodes for the channel -
// not the channel as a whole, since different subscribers to the same channel can have
// completely different retention preferences. Evicting a subscriber's own flag never deletes the
// shared file outright; the file is only removed once no subscriber anywhere still wants it (see
// cleanupOrphanedFile). Run after each scheduled refresh, and callable directly.
func (p *podcasts) RunRetention(ctx context.Context) error {
	subs, err := p.ds.PodcastSubscription(ctx).GetAll()
	if err != nil {
		return err
	}
	var firstErr error
	for _, sub := range subs {
		if !sub.HasRetentionLimit() {
			continue
		}
		if err := p.runSubscriptionRetention(ctx, sub); err != nil {
			log.Error(ctx, "Error enforcing podcast retention", "subscription", sub.ID, "channel", sub.ChannelID, "user", sub.UserID, err)
			if firstErr == nil {
				firstErr = err
			}
		}
	}
	return firstErr
}

func (p *podcasts) runSubscriptionRetention(ctx context.Context, sub model.PodcastSubscription) error {
	episodes, err := p.ds.PodcastEpisode(ctx).GetDownloadedForUser(sub.UserID, sub.ChannelID)
	if err != nil {
		return err
	}
	for _, ep := range retentionCandidates(sub, episodes) {
		pinned, err := p.isPinnedByPlaylist(ctx, ep.ID)
		if err != nil {
			log.Error(ctx, "Error checking playlist membership before retention eviction", "subscription", sub.ID, "episode", ep.ID, err)
			continue
		}
		if pinned {
			log.Debug(ctx, "Skipping retention eviction for episode referenced by a playlist", "subscription", sub.ID, "episode", ep.ID)
			continue
		}
		if err := p.ds.PodcastEpisode(ctx).SetDownloaded(sub.UserID, false, ep.ID); err != nil {
			log.Error(ctx, "Error clearing downloaded flag during retention eviction", "subscription", sub.ID, "episode", ep.ID, err)
			continue
		}
		if err := p.cleanupOrphanedFile(ctx, ep.ID); err != nil {
			log.Error(ctx, "Error cleaning up podcast episode file during retention", "episode", ep.ID, err)
		}
	}
	return nil
}

// isPinnedByPlaylist reports whether an episode is referenced by any
// playlist, exempting it from automatic retention cleanup - if it was
// deliberately queued for listening, retention shouldn't silently delete it.
func (p *podcasts) isPinnedByPlaylist(ctx context.Context, episodeID string) (bool, error) {
	playlists, err := p.ds.Playlist(ctx).GetPlaylists(episodeID)
	if err != nil {
		return false, err
	}
	return len(playlists) > 0, nil
}

// retentionCandidates returns which of a subscriber's own downloaded episodes for one channel
// (already sorted newest-first by publish date) exceed that subscription's own retention policy.
// Each configured limit (0 = unlimited) is evaluated independently; an episode beyond the count
// cap, older than the day cap, or beyond the cumulative storage budget is a candidate for
// eviction from just this subscriber's own list.
func retentionCandidates(sub model.PodcastSubscription, episodes model.PodcastEpisodes) model.PodcastEpisodes {
	var cutoff time.Time
	if sub.RetentionDays > 0 {
		cutoff = time.Now().AddDate(0, 0, -sub.RetentionDays)
	}
	maxStorageBytes := int64(sub.MaxStorageMB) * 1024 * 1024

	var candidates model.PodcastEpisodes
	var cumulativeSize int64
	for i, ep := range episodes {
		cumulativeSize += ep.Size
		exceedsCount := sub.RetentionCount > 0 && i >= sub.RetentionCount
		exceedsAge := sub.RetentionDays > 0 && ep.PublishDate != nil && ep.PublishDate.Before(cutoff)
		exceedsStorage := sub.MaxStorageMB > 0 && cumulativeSize > maxStorageBytes
		if exceedsCount || exceedsAge || exceedsStorage {
			candidates = append(candidates, ep)
		}
	}
	return candidates
}
