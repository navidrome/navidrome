package host

import "context"

type ScrobbleList struct {
	Scrobbles     []ScrobbleRef `json:"scrobbles"`
	NextTimestamp *int64        `json:"nextTimestamp,omitempty"`
}

type ScrobbleRef struct {
	ID             int64  `json:"id"`
	MediaFileID    string `json:"mediaFileId"`
	SubmissionTime int64  `json:"submissionTime"`
}

type ScrobbleOptions struct {
	FromTimestamp *int64 `json:"fromTimestamp,omitempty"`
	ToTimestamp   *int64 `json:"toTimestamp,omitempty"`
	MaxItems      int    `json:"maxItems"`
}

// ScrobbleRetrieverService allows a plugin to retrieve scrobbles for one or more authorized users.
// It will only provide the media_file ID and submission time, which can be combined with the MatcherService
// to fetch deduped tracks
//
//nd:hostservice name=ScrobbleRetriever permission=scrobbleRetriever
type ScrobbleRetrieverService interface {
	// GetFirstTimestamp returns the unix timestamp of the oldest scrobble for the user
	//nd:hostfunc
	GetFirstTimestamp(ctx context.Context, username string) (*int64, error)

	// GetLastTimestamp returns the unix timestamp of the most recent scrobble for the user
	//nd:hostfunc
	GetLastTimestamp(ctx context.Context, username string) (*int64, error)

	//nd:hostfunc
	GetScrobbles(ctx context.Context, username string, options ScrobbleOptions) (*ScrobbleList, error)
}
