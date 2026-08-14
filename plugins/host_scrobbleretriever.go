package plugins

import (
	"context"
	"fmt"

	"github.com/Masterminds/squirrel"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
	"github.com/navidrome/navidrome/plugins/host"
)

// maxScrobbleItems caps how many scrobbles a single GetScrobbles call can return.
const maxScrobbleItems = 5000

type scrobbleRetrieverServiceImpl struct {
	ds    model.DataStore
	users userAccess
}

func newScrobbleRetrieverService(ds model.DataStore, users userAccess) host.ScrobbleRetrieverService {
	return &scrobbleRetrieverServiceImpl{
		ds:    ds,
		users: users,
	}
}

func (s *scrobbleRetrieverServiceImpl) getUserContext(ctx context.Context, username string) (context.Context, error) {
	usr, err := s.users.resolve(ctx, s.ds, username)
	if err != nil {
		return nil, fmt.Errorf("scrobbleRetriever: %w", err)
	}

	ctx = request.WithUser(ctx, *usr)
	return ctx, nil
}

func (s *scrobbleRetrieverServiceImpl) getFirstLastScrobble(ctx context.Context, username string, order string) (*int64, error) {
	ctx, err := s.getUserContext(ctx, username)
	if err != nil {
		return nil, err
	}

	scrobbles, err := s.ds.Scrobble(ctx).GetAll(model.QueryOptions{Sort: "submission_time", Order: order, Max: 1})
	if err != nil {
		return nil, err
	}

	if len(scrobbles) == 0 {
		return nil, nil
	}

	return &scrobbles[0].SubmissionTime, nil
}

func (s *scrobbleRetrieverServiceImpl) GetFirstTimestamp(ctx context.Context, username string) (*int64, error) {
	return s.getFirstLastScrobble(ctx, username, "ASC")
}

func (s *scrobbleRetrieverServiceImpl) GetLastTimestamp(ctx context.Context, username string) (*int64, error) {
	return s.getFirstLastScrobble(ctx, username, "DESC")
}

func (s *scrobbleRetrieverServiceImpl) GetScrobbles(ctx context.Context, username string, options host.ScrobbleOptions) ([]host.ScrobbleRef, *host.ScrobbleOptions, error) {
	ctx, err := s.getUserContext(ctx, username)
	if err != nil {
		return nil, nil, err
	}

	if options.MaxItems < 1 || options.MaxItems > maxScrobbleItems {
		options.MaxItems = maxScrobbleItems
	}
	options.Offset = max(options.Offset, 0)

	order := "ASC"
	if options.Descending {
		order = "DESC"
	}

	// Fetch one more item than requested. The last item is the next timestamp to fetch
	lookahead := options.MaxItems + 1

	scrobbles, err := s.ds.Scrobble(ctx).GetAll(model.QueryOptions{
		Max:     lookahead,
		Filters: scrobbleRangeFilters(options.FromTimestamp, options.ToTimestamp),
		// The id tiebreak makes the order of equal timestamps stable, which is what
		// lets Offset skip exactly the ties already returned
		Sort:   "scrobbles.submission_time, scrobbles.id",
		Order:  order,
		Offset: options.Offset,
	})
	if err != nil {
		return nil, nil, err
	}

	var next *host.ScrobbleOptions
	targetLen := len(scrobbles)

	if len(scrobbles) == lookahead {
		nextTimestamp := scrobbles[lookahead-1].SubmissionTime
		targetLen = lookahead - 1

		ties := 0
		for i := targetLen - 1; i >= 0; i-- {
			if scrobbles[i].SubmissionTime != nextTimestamp {
				break
			}
			ties++
		}

		// Every scrobble in this page shares the timestamp, so the ties skipped by the
		// incoming offset are still ahead of us and must be carried over
		if ties == targetLen {
			ties += options.Offset
		}

		advanced := options
		advanced.Offset = ties
		if options.Descending {
			advanced.ToTimestamp = &nextTimestamp
		} else {
			advanced.FromTimestamp = &nextTimestamp
		}
		next = &advanced
	}

	scrobbleRefs := make([]host.ScrobbleRef, targetLen)
	for idx, scrobble := range scrobbles[:targetLen] {
		scrobbleRefs[idx] = host.ScrobbleRef{
			ID:             scrobble.ID,
			MediaFileID:    scrobble.MediaFileID,
			SubmissionTime: scrobble.SubmissionTime,
		}
	}

	return scrobbleRefs, next, nil
}

func (s *scrobbleRetrieverServiceImpl) GetScrobbleCount(ctx context.Context, username string, options host.ScrobbleCountOptions) (int64, error) {
	ctx, err := s.getUserContext(ctx, username)
	if err != nil {
		return 0, err
	}

	return s.ds.Scrobble(ctx).CountAll(model.QueryOptions{
		Filters: scrobbleRangeFilters(options.FromTimestamp, options.ToTimestamp),
	})
}

func scrobbleRangeFilters(from, to *int64) squirrel.And {
	var filters squirrel.And
	if from != nil {
		filters = append(filters, squirrel.GtOrEq{"scrobbles.submission_time": *from})
	}
	if to != nil {
		filters = append(filters, squirrel.LtOrEq{"scrobbles.submission_time": *to})
	}
	return filters
}

var _ host.ScrobbleRetrieverService = (*scrobbleRetrieverServiceImpl)(nil)
