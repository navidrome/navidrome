package plugins

import (
	"context"
	"fmt"

	"github.com/Masterminds/squirrel"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
	"github.com/navidrome/navidrome/plugins/host"
)

type scrobbleRetrieverServiceImpl struct {
	ds    model.DataStore
	users userAccess
}

func newScrobbleRetreverService(ds model.DataStore, users userAccess) host.ScrobbleRetrieverService {
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

	if options.MaxItems == 0 {
		options.MaxItems = 5000
	}

	// Fetch one more item than requested. The last item is the next timestamp to fetch
	options.MaxItems += 1

	var filters squirrel.Sqlizer
	if options.FromTimestamp != nil {
		filters = squirrel.GtOrEq{"submission_time": *options.FromTimestamp}
	}

	if options.ToTimestamp != nil {
		filters = squirrel.And{filters, squirrel.LtOrEq{"submission_time": *options.ToTimestamp}}
	}

	var order string
	if options.ToTimestamp != nil && options.FromTimestamp == nil {
		order = "DESC"
	} else {
		order = "ASC"
	}

	scrobbles, err := s.ds.Scrobble(ctx).GetAll(model.QueryOptions{
		Max:     options.MaxItems,
		Filters: filters,
		Sort:    "submission_time",
		Order:   order,
	})

	if err != nil {
		return nil, err
	}

	var nextTimestamp *int64

	if len(scrobbles) == options.MaxItems {
		nextTimestamp = &scrobbles[options.MaxItems-1].SubmissionTime
	}

	scrobbleRefs := make([]host.ScrobbleRef, options.MaxItems-1)
	for idx := range scrobbleRefs {
		scrobbleRefs[idx].ID = scrobbles[idx].ID
		scrobbleRefs[idx].MediaFileID = scrobbles[idx].MediaFileID
		scrobbleRefs[idx].SubmissionTime = scrobbles[idx].SubmissionTime
	}

	response := host.ScrobbleList{
		Scrobbles:     scrobbleRefs,
		NextTimestamp: nextTimestamp,
	}

	return &response, nil
}

var _ host.ScrobbleRetrieverService = (*scrobbleRetrieverServiceImpl)(nil)
