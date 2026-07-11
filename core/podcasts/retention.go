package podcasts

import (
	"context"
	"time"

	. "github.com/Masterminds/squirrel"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
)

// RunRetention enforces each channel's RetentionCount, RetentionDays and
// MaxStorageMB policies (0 = unlimited), deleting the oldest downloaded
// episodes beyond whichever limits are configured. Run after each scheduled
// refresh, and callable directly.
func (p *podcasts) RunRetention(ctx context.Context) error {
	channels, err := p.ds.PodcastChannel(ctx).GetAll()
	if err != nil {
		return err
	}
	var firstErr error
	for _, channel := range channels {
		if channel.RetentionCount <= 0 && channel.RetentionDays <= 0 && channel.MaxStorageMB <= 0 {
			continue
		}
		if err := p.runChannelRetention(ctx, channel); err != nil {
			log.Error(ctx, "Error enforcing podcast retention", "channel", channel.ID, err)
			if firstErr == nil {
				firstErr = err
			}
		}
	}
	return firstErr
}

func (p *podcasts) runChannelRetention(ctx context.Context, channel model.PodcastChannel) error {
	episodes, err := p.ds.PodcastEpisode(ctx).GetAll(model.QueryOptions{
		Filters: And{Eq{"channel_id": channel.ID}, Eq{"download_status": string(model.PodcastEpisodeDownloaded)}},
		Sort:    "publish_date",
		Order:   "desc",
	})
	if err != nil {
		return err
	}
	for _, ep := range retentionCandidates(channel, episodes) {
		if err := p.deleteEpisodeFile(ctx, &ep); err != nil {
			log.Error(ctx, "Error deleting podcast episode during retention cleanup", "channel", channel.ID, "episode", ep.ID, err)
		}
	}
	return nil
}

// retentionCandidates returns which of a channel's downloaded episodes
// (already sorted newest-first by publish date) exceed the channel's
// retention policy. Each configured limit (0 = unlimited) is evaluated
// independently; an episode beyond the count cap, older than the day cap,
// or beyond the cumulative storage budget is a candidate for deletion.
func retentionCandidates(channel model.PodcastChannel, episodes model.PodcastEpisodes) model.PodcastEpisodes {
	var cutoff time.Time
	if channel.RetentionDays > 0 {
		cutoff = time.Now().AddDate(0, 0, -channel.RetentionDays)
	}
	maxStorageBytes := int64(channel.MaxStorageMB) * 1024 * 1024

	var candidates model.PodcastEpisodes
	var cumulativeSize int64
	for i, ep := range episodes {
		cumulativeSize += ep.Size
		exceedsCount := channel.RetentionCount > 0 && i >= channel.RetentionCount
		exceedsAge := channel.RetentionDays > 0 && ep.PublishDate != nil && ep.PublishDate.Before(cutoff)
		exceedsStorage := channel.MaxStorageMB > 0 && cumulativeSize > maxStorageBytes
		if exceedsCount || exceedsAge || exceedsStorage {
			candidates = append(candidates, ep)
		}
	}
	return candidates
}
