package host

import "context"

// ScrobbleList is a list of scrobbles, plus an optional timestamp
// that can be used as a cursor for the next fetch
type ScrobbleList struct {
	// The scrobbles in a given range
	Scrobbles []ScrobbleRef `json:"scrobbles"`
	// If additional items are available, the timestamp of the next scrobble to fetch
	NextTimestamp *int64 `json:"nextTimestamp,omitempty"`
}

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
	// The maximum number of items to retrieve. This is capped at 5000, the
	// default if not specified
	MaxItems int `json:"maxItems"`
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

	// GetScrobbles returns scrobbles for a user.
	//
	// Parameters:
	//   - username: the user to query for scrobbles
	//   - options.FromTimestamp: If specified, the first UNIX timestamp to start fetching scrobbles (inclusive). Otherwise, start from the first scrobble
	//   - options.ToTimestamp: If specified, the last UNIX timestamp to fetch (inclusive). Otherwise, end at the last scrobble
	//   - options.MaxItems: The maximum number of items to retrieve. The maximum value (and default) if not specified is 5000
	//
	// Returns:
	//   - Scrobbles: A list of scrobbles within the constraints given (if any). The order
	//     of the items depends on the options: if ToTimestamp is specified AND
	//     FromTImestamp is not specified, the order is in descending submission time.
	//     Otherwise, the scrobbles are returned in ascending submission time.
	//   - NextTimestamp: If there are additional items to retrieve in the range, the timestamp
	//     of the next scrobble that would be retrieved in the order (asc or desc)
	//nd:hostfunc
	GetScrobbles(ctx context.Context, username string, options ScrobbleOptions) (*ScrobbleList, error)

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
