package external

import (
	"context"

	"github.com/navidrome/navidrome/core/matcher"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/server/events"
)

// NewProviderWithContext stops the background refresh queues when ctx ends, so a spec's queue
// cannot call its agents after the spec is over.
func NewProviderWithContext(ctx context.Context, ds model.DataStore, ag Agents, m *matcher.Matcher, broker events.Broker) Provider {
	return newProvider(ctx, ds, ag, m, broker)
}

func QueuedAlbumRefreshes(p Provider) int  { return len(p.(*provider).albumQueue) }
func QueuedArtistRefreshes(p Provider) int { return len(p.(*provider).artistQueue) }
