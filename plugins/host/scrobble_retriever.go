package host

import "context"

// ScrobbleRef represents one instance of a scrobble (instance id, file id, submission time)
type ScrobbleRef struct {
	// The ID of the scrobble. Useful if duplicate scrobbles happen for the same time
	ID int64 `json:"id"`
	// The ID of the MediaFile submitted at this time
	MediaFileID string `json:"mediaFileId"`
	// The UNIX timestamp this scrobble was submitted
	SubmissionTime int64 `json:"submissionTime"`
}

// ScrobbleOptions carries optional parameters for retrieving user scrobbles
type ScrobbleOptions struct {
	// The starting unix timestamp to query for scrobbles (inclusive).
	// If not specified, start from the first scrobble
	FromTimestamp *int64 `json:"fromTimestamp,omitempty"`
	// The ending unix timestamp to query for scrobbles (inclusive).
	// If not specified, go up to the last scrobble
	ToTimestamp *int64 `json:"toTimestamp,omitempty"`
	// If true, return scrobbles from newest to oldest. Defaults to oldest first
	Descending bool `json:"descending"`
	// The maximum number of items to retrieve. This is capped at 5000, the
	// default if not specified
	MaxItems int `json:"maxItems"`
	// Pagination state managed by GetScrobbles; only meaningful on the options it returns.
	// Never set it or combine it with your own timestamps — scrobbles may be silently skipped
	Offset int `json:"offset,omitempty"`
}

// ScrobbleCountOptions carries optional parameters for counting user scrobbles
type ScrobbleCountOptions struct {
	// The starting unix timestamp to query for scrobbles (inclusive).
	// If not specified, start from the first scrobble
	FromTimestamp *int64 `json:"fromTimestamp,omitempty"`
	// The ending unix timestamp to query for scrobbles (inclusive).
	// If not specified, go up to the last scrobble
	ToTimestamp *int64 `json:"toTimestamp,omitempty"`
}

// ScrobbleRetrieverService allows a plugin to retrieve scrobbles for one or more authorized users.
// It will only provide the media_file ID and submission time, which can be combined with the MatcherService
// to fetch deduped tracks
//
//nd:hostservice name=ScrobbleRetriever permission=scrobbleRetriever
type ScrobbleRetrieverService interface {
	// GetFirstTimestamp returns the unix timestamp of the oldest scrobble for the user.
	// If the user has no scrobbles, returns nil
	//nd:hostfunc
	GetFirstTimestamp(ctx context.Context, username string) (*int64, error)

	// GetLastTimestamp returns the unix timestamp of the most recent scrobble for the user
	// If the user has no scrobbles, return nil
	//nd:hostfunc
	GetLastTimestamp(ctx context.Context, username string) (*int64, error)

	// GetScrobbles returns one page of scrobbles for a user.
	//
	// Parameters:
	//   - username: the user to query for scrobbles
	//   - options.FromTimestamp: If specified, the first UNIX timestamp to start fetching scrobbles (inclusive). Otherwise, start from the first scrobble
	//   - options.ToTimestamp: If specified, the last UNIX timestamp to fetch (inclusive). Otherwise, end at the last scrobble
	//   - options.Descending: If true, order from newest to oldest. Otherwise, oldest to newest
	//   - options.MaxItems: The maximum number of items to retrieve. The maximum value (and default) if not specified is 5000
	//   - options.Offset: Pagination state; only valid as received on the options returned by a previous call. Never set it manually
	//
	// Returns:
	//   - scrobbles: The scrobbles in the requested range, ordered by submission time
	//     (ties broken by scrobble ID) in the direction given by options.Descending
	//   - next: The options for the following page, or nil once no scrobbles remain.
	//     Pass it back to GetScrobbles unchanged and repeat until it is nil. It carries an
	//     adjusted FromTimestamp/ToTimestamp, so keep a copy if you still need the original range
	//nd:hostfunc
	GetScrobbles(ctx context.Context, username string, options ScrobbleOptions) (scrobbles []ScrobbleRef, next *ScrobbleOptions, err error)

	// GetScrobbleCount returns the number of scrobbles for a user in a given range
	//
	// Parameters:
	//   - username: the user to query for scrobbles
	//   - options.FromTimestamp: If specified, the first UNIX timestamp to start fetching scrobbles (inclusive). Otherwise, start from the first scrobble
	//   - options.ToTimestamp: If specified, the last UNIX timestamp to fetch (inclusive). Otherwise, end at the last scrobble
	//
	// Returns:
	//   - the number of scrobbles in the given range, or 0
	//nd:hostfunc
	GetScrobbleCount(ctx context.Context, username string, options ScrobbleCountOptions) (int64, error)
}
