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

func (s *scrobbleRetrieverServiceImpl) GetScrobbles(ctx context.Context, username string, options host.ScrobbleOptions) (*host.ScrobbleList, error) {
	ctx, err := s.getUserContext(ctx, username)
	if err != nil {
		return nil, err
	}

	if options.MaxItems < 1 || options.MaxItems > maxScrobbleItems {
		options.MaxItems = maxScrobbleItems
	}

	if options.Cursor < 0 {
		options.Cursor = 0
	}

	// Fetch one more item than requested. The last item is the next timestamp to fetch
	options.MaxItems += 1

	order := "ASC"
	if options.Descending {
		order = "DESC"
	}

	scrobbles, err := s.ds.Scrobble(ctx).GetAll(model.QueryOptions{
		Max:     options.MaxItems,
		Filters: scrobbleRangeFilters(options.FromTimestamp, options.ToTimestamp),
		// The id tiebreak makes the order of equal timestamps stable, which is what
		// lets Cursor skip exactly the ties already returned
		Sort:   "scrobbles.submission_time, scrobbles.id",
		Order:  order,
		Offset: options.Cursor,
	})

	if err != nil {
		return nil, err
	}

	if len(scrobbles) == 0 {
		return &host.ScrobbleList{Scrobbles: []host.ScrobbleRef{}}, nil
	}

	cursor := 0

	var nextTimestamp *int64
	var targetLen int

	if len(scrobbles) == options.MaxItems {
		nextTimestamp = &scrobbles[options.MaxItems-1].SubmissionTime
		targetLen = options.MaxItems - 1

		lastTimestamp := scrobbles[len(scrobbles)-1].SubmissionTime
		for i := len(scrobbles) - 2; i >= 0; i-- {
			if scrobbles[i].SubmissionTime != lastTimestamp {
				break
			}

			cursor += 1
		}

		// In this case, every scrobble in this query is the same timestamp
		// We should continue from the previous cursor
		if cursor == len(scrobbles)-1 {
			cursor += options.Cursor
		}
	} else {
		targetLen = len(scrobbles)
	}

	scrobbleRefs := make([]host.ScrobbleRef, targetLen)

	for idx := range targetLen {
		scrobbleRefs[idx].ID = scrobbles[idx].ID
		scrobbleRefs[idx].MediaFileID = scrobbles[idx].MediaFileID
		scrobbleRefs[idx].SubmissionTime = scrobbles[idx].SubmissionTime
	}

	response := host.ScrobbleList{
		Scrobbles:     scrobbleRefs,
		NextTimestamp: nextTimestamp,
		Cursor:        cursor,
	}

	return &response, nil
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
