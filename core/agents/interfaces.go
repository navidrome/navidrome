package agents

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/gohugoio/hashstructure"
	"github.com/navidrome/navidrome/model"
)

type Constructor func(ds model.DataStore) Interface

type Interface interface {
	AgentName() string
}

// AlbumInfo contains album metadata (no images)
type AlbumInfo struct {
	Name        string
	MBID        string
	Description string
	URL         string
}

type Artist struct {
	ID   string
	Name string
	MBID string
}

type ExternalImage struct {
	URL  string
	Size int
}

type Song struct {
	ID        string
	Name      string
	MBID      string
	ISRC      string
	Artists   []Artist
	Album     string
	AlbumMBID string
	Duration  uint32 // Duration in milliseconds, 0 means unknown
}

// Equals reports strict whole-value equality, used to dedup identical input songs. It hashes
// rather than comparing with ==, which the Artists slice makes illegal.
func (s Song) Equals(other Song) bool {
	h1, _ := hashstructure.Hash(s, nil)
	h2, _ := hashstructure.Hash(other, nil)
	return h1 == h2
}

var (
	// ErrNotFound means the provider answered and had nothing. Return the underlying error
	// for a fault instead, or callers that back off on faults will treat it as definitive.
	ErrNotFound = errors.New("not found")

	// ErrRetryLater means the provider is temporarily unavailable or throttling us.
	ErrRetryLater = errors.New("retry later")
)

// RetryLaterError is an ErrRetryLater carrying the server-requested delay (0 = unspecified).
type RetryLaterError struct {
	RetryIn time.Duration
}

func (e *RetryLaterError) Error() string {
	if e.RetryIn > 0 {
		return fmt.Sprintf("retry later (in %s)", e.RetryIn)
	}
	return "retry later"
}

func (e *RetryLaterError) Is(target error) bool { return target == ErrRetryLater }

// maxRetryIn caps a requested delay, so a bogus value cannot park a provider indefinitely.
const maxRetryIn = time.Hour
const maxRetryInSeconds = int(maxRetryIn / time.Second)

// NewRetryLater returns a RetryLaterError asking for d, clamped to [0, maxRetryIn].
func NewRetryLater(d time.Duration) *RetryLaterError {
	return &RetryLaterError{RetryIn: min(max(d, 0), maxRetryIn)}
}

// RetryLaterFromSeconds builds a RetryLaterError from a delay in seconds, as sent by plugins
// and HTTP headers. An empty or unparseable s means the delay is unspecified.
func RetryLaterFromSeconds(s string) *RetryLaterError {
	// Clamped in seconds: scaling first would let a huge value wrap around to a tiny delay.
	if secs, err := strconv.Atoi(s); err == nil && secs > 0 {
		return &RetryLaterError{RetryIn: time.Duration(min(secs, maxRetryInSeconds)) * time.Second}
	}
	return &RetryLaterError{}
}

// RetryIn extracts the server-requested delay from err, if one is attached.
func RetryIn(err error) (time.Duration, bool) {
	var rle *RetryLaterError
	if errors.As(err, &rle) {
		return rle.RetryIn, true
	}
	return 0, false
}

// AlbumInfoRetriever provides album info (no images)
type AlbumInfoRetriever interface {
	GetAlbumInfo(ctx context.Context, name, artist, mbid string) (*AlbumInfo, error)
}

// AlbumImageRetriever provides album images
type AlbumImageRetriever interface {
	GetAlbumImages(ctx context.Context, name, artist, mbid string) ([]ExternalImage, error)
}

type ArtistMBIDRetriever interface {
	GetArtistMBID(ctx context.Context, id string, name string) (string, error)
}

type ArtistURLRetriever interface {
	GetArtistURL(ctx context.Context, id, name, mbid string) (string, error)
}

type ArtistBiographyRetriever interface {
	GetArtistBiography(ctx context.Context, id, name, mbid string) (string, error)
}

type ArtistSimilarRetriever interface {
	GetSimilarArtists(ctx context.Context, id, name, mbid string, limit int) ([]Artist, error)
}

type ArtistImageRetriever interface {
	GetArtistImages(ctx context.Context, id, name, mbid string) ([]ExternalImage, error)
}

type ArtistTopSongsRetriever interface {
	GetArtistTopSongs(ctx context.Context, id, artistName, mbid string, count int) ([]Song, error)
}

// SimilarSongsByTrackRetriever provides similar songs based on a specific track
type SimilarSongsByTrackRetriever interface {
	// GetSimilarSongsByTrack returns songs similar to the given track.
	// Parameters:
	//   - id: local mediafile ID
	//   - name: track title
	//   - artist: artist name
	//   - mbid: MusicBrainz recording ID (may be empty)
	//   - count: maximum number of results
	GetSimilarSongsByTrack(ctx context.Context, id, name, artist, mbid string, count int) ([]Song, error)
}

// SimilarSongsByAlbumRetriever provides similar songs based on an album
type SimilarSongsByAlbumRetriever interface {
	// GetSimilarSongsByAlbum returns songs similar to tracks on the given album.
	// Parameters:
	//   - id: local album ID
	//   - name: album name
	//   - artist: album artist name
	//   - mbid: MusicBrainz release ID (may be empty)
	//   - count: maximum number of results
	GetSimilarSongsByAlbum(ctx context.Context, id, name, artist, mbid string, count int) ([]Song, error)
}

// SimilarSongsByArtistRetriever provides similar songs based on an artist
type SimilarSongsByArtistRetriever interface {
	// GetSimilarSongsByArtist returns songs similar to the artist's catalog.
	// Parameters:
	//   - id: local artist ID
	//   - name: artist name
	//   - mbid: MusicBrainz artist ID (may be empty)
	//   - count: maximum number of results
	GetSimilarSongsByArtist(ctx context.Context, id, name, mbid string, count int) ([]Song, error)
}

var Map map[string]Constructor

func Register(name string, init Constructor) {
	if Map == nil {
		Map = make(map[string]Constructor)
	}
	Map[name] = init
}
