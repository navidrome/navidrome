// Package rustsearch provides the persistent Rust/Tantivy search companion.
// SQLite FTS remains the fallback while this derived index starts or rebuilds.
package rustsearch

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/navidrome/navidrome/core/auth"
	"github.com/navidrome/navidrome/core/eventbus"
	"github.com/navidrome/navidrome/core/rustworker"
	"github.com/navidrome/navidrome/core/searchworker"
	"github.com/navidrome/navidrome/core/searchworker/gen"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/query"
)

const (
	protocolVersion        = 1
	MaxResults             = 500
	indexBatchSize         = 5000
	freshnessCheckInterval = 10 * time.Second
	searchRequestTimeout   = 5 * time.Second
	indexRequestTimeout    = 60 * time.Second
	// Prefer a full rebuild when the delta is large enough that chunked
	// upsert/delete commits would spend more time than a single replacement.
	maxIncrementalRatio = 0.25
)

var ErrNotReady = errors.New("Rust search index is not ready")

type document struct {
	Key        string   `json:"key"`
	ID         string   `json:"id"`
	Kind       string   `json:"kind"`
	LibraryIDs []uint64 `json:"library_ids,omitempty"`
	Primary    string   `json:"primary"`
	Secondary  string   `json:"secondary,omitempty"`
}

type request struct {
	Op         string       `json:"op"`
	Documents  []document   `json:"documents,omitempty"`
	Keys       []string     `json:"keys,omitempty"`
	Query      string       `json:"query,omitempty"`
	LibraryIDs []uint64     `json:"library_ids,omitempty"`
	Searches   []searchSpec `json:"searches,omitempty"`
	Values     []string     `json:"values,omitempty"`
}

type searchSpec struct {
	Kind   string `json:"kind"`
	Offset int    `json:"offset,omitempty"`
	Limit  int    `json:"limit,omitempty"`
}

type hit struct {
	ID    string  `json:"id"`
	Score float32 `json:"score"`
}

type searchGroup struct {
	Kind string `json:"kind"`
	Hits []hit  `json:"hits"`
}

type response struct {
	Protocol   int           `json:"protocol"`
	OK         bool          `json:"ok"`
	Groups     []searchGroup `json:"groups"`
	Indexed    uint64        `json:"indexed"`
	Error      string        `json:"error"`
	Normalized string        `json:"normalized,omitempty"`
}

type SearchLimits struct {
	Offset int
	Limit  int
}

type SearchResults struct {
	SongIDs   []string
	AlbumIDs  []string
	ArtistIDs []string
}

type Engine struct {
	gate         sync.RWMutex
	grpc         gen.SearchClient
	ready        atomic.Bool
	building     atomic.Bool
	generation   atomic.Int64
	nextCheck    atomic.Int64
	indexed      atomic.Uint64
	allowInTests atomic.Bool
}

func Available() bool {
	_, err := searchworker.Resolve()
	return err == nil
}

func New() *Engine {
	return &Engine{}
}

func (e *Engine) EnableForTests() {
	if e != nil {
		e.allowInTests.Store(true)
	}
}

// Shutdown stops the Rust search worker and releases the on-disk index lock.
func (e *Engine) Shutdown() {
	if e == nil {
		return
	}
	e.allowInTests.Store(false)
	e.WaitIdle()
	e.stopWorker()
}

// WaitIdle blocks until any in-flight index build or refresh completes.
func (e *Engine) WaitIdle() {
	if e == nil {
		return
	}
	for e.building.Load() {
		time.Sleep(time.Millisecond)
	}
}

func (e *Engine) skipBackgroundWork() bool {
	return e == nil || (testing.Testing() && !e.allowInTests.Load())
}

// ListenForScans subscribes to library scan completion so the Tantivy index
// refreshes from the event stream instead of polling other systems' clocks.
// The returned function unsubscribes; callers must invoke it before shutdown.
func (e *Engine) ListenForScans(ds model.DataStore) func() {
	if e == nil || ds == nil {
		return func() {}
	}
	return eventbus.Get().Subscribe(eventbus.TopicScanCompleted, func(ctx context.Context, _ eventbus.Event) {
		e.nextCheck.Store(0)
		e.RefreshIfStale(ctx, ds)
	})
}

func (e *Engine) Ready() bool {
	return e != nil && e.ready.Load()
}

func (e *Engine) SearchAll(ctx context.Context, query string, libraryIDs []int, songs, albums, artists SearchLimits) (SearchResults, error) {
	if !e.Ready() {
		return SearchResults{}, ErrNotReady
	}
	resp, err := e.roundTrip(ctx, request{
		Op:         "search_all",
		Query:      query,
		LibraryIDs: libraryScope(libraryIDs),
		Searches: []searchSpec{
			{Kind: "song", Offset: songs.Offset, Limit: songs.Limit},
			{Kind: "album", Offset: albums.Offset, Limit: albums.Limit},
			{Kind: "artist", Offset: artists.Offset, Limit: artists.Limit},
		},
	})
	if err != nil {
		return SearchResults{}, err
	}
	return decodeSearchGroups(resp.Groups)
}

func decodeSearchGroups(groups []searchGroup) (SearchResults, error) {
	var results SearchResults
	seen := 0
	for _, group := range groups {
		ids := make([]string, len(group.Hits))
		for i, hit := range group.Hits {
			ids[i] = hit.ID
		}
		switch group.Kind {
		case "song":
			results.SongIDs = ids
			seen |= 1
		case "album":
			results.AlbumIDs = ids
			seen |= 2
		case "artist":
			results.ArtistIDs = ids
			seen |= 4
		}
	}
	if seen != 7 {
		return SearchResults{}, errors.New("Rust search_all response is missing a result group")
	}
	return results, nil
}

func libraryScope(libraryIDs []int) []uint64 {
	scope := make([]uint64, 0, len(libraryIDs))
	for _, id := range libraryIDs {
		if id > 0 {
			scope = append(scope, uint64(id))
		}
	}
	return scope
}

// RefreshIfStale checks scan generations at a bounded cadence. Searches keep
// using the old index until a replacement or incremental sync commits.
func (e *Engine) RefreshIfStale(ctx context.Context, ds model.DataStore) {
	if e.skipBackgroundWork() {
		return
	}
	if e == nil || e.building.Load() {
		return
	}
	now := time.Now()
	next := e.nextCheck.Load()
	if next > now.UnixNano() || !e.nextCheck.CompareAndSwap(next, now.Add(freshnessCheckInterval).UnixNano()) {
		return
	}
	adminCtx := auth.WithAdminUser(context.WithoutCancel(ctx), ds)
	libraries, err := ds.Library(adminCtx).GetAll()
	if err != nil {
		log.Debug(ctx, "Rust search freshness check failed", err)
		return
	}
	ready := e.Ready()
	generation := e.generation.Load()
	if !searchIndexStale(ready, generation, libraries) {
		return
	}
	refresh := func() {
		var err error
		if ready && generation > 0 {
			err = e.RefreshIncremental(adminCtx, ds, generation)
			if err != nil {
				log.Debug(adminCtx, "Rust search incremental refresh failed; falling back to full rebuild", err)
				err = e.Rebuild(adminCtx, ds)
			}
		} else {
			err = e.Rebuild(adminCtx, ds)
		}
		if err != nil {
			log.Warn("Rust search index rebuild failed; SQLite search remains active", err)
		}
	}
	if e.allowInTests.Load() {
		refresh()
		return
	}
	go refresh()
}

// RefreshIncremental applies scan deltas with upsert/delete instead of rebuilding
// the whole in-RAM Tantivy index. Go remains the control tower: it reads SQLite,
// builds documents, and decides when to fall back to Rebuild.
func (e *Engine) RefreshIncremental(ctx context.Context, ds model.DataStore, sinceGeneration int64) error {
	if !e.building.CompareAndSwap(false, true) {
		return nil
	}
	defer e.building.Store(false)
	if !e.ready.Load() || sinceGeneration <= 0 {
		return ErrNotReady
	}

	ctx = auth.WithAdminUser(ctx, ds)
	libraries, err := ds.Library(ctx).GetAll()
	if err != nil {
		return fmt.Errorf("loading libraries for Rust search: %w", err)
	}
	if !searchIndexStale(true, sinceGeneration, libraries) {
		return nil
	}

	since := time.Unix(0, sinceGeneration)
	expected, err := expectedSearchDocuments(ctx, ds)
	if err != nil {
		return err
	}
	indexedBefore := int64(e.indexed.Load())
	if preferFullRebuild(indexedBefore, abs64(expected-indexedBefore)) {
		return e.rebuildLocked(ctx, ds, libraries)
	}

	upserts := make([]document, 0, indexBatchSize)
	deletes := make([]string, 0, indexBatchSize)
	changed := int64(0)
	var lastIndexed uint64
	flushUpserts := func() error {
		if len(upserts) == 0 {
			return nil
		}
		resp, err := e.roundTrip(ctx, request{Op: "upsert", Documents: upserts})
		if err != nil {
			return err
		}
		lastIndexed = resp.Indexed
		upserts = upserts[:0]
		return nil
	}
	flushDeletes := func() error {
		if len(deletes) == 0 {
			return nil
		}
		resp, err := e.roundTrip(ctx, request{Op: "delete", Keys: deletes})
		if err != nil {
			return err
		}
		lastIndexed = resp.Indexed
		deletes = deletes[:0]
		return nil
	}
	queueUpsert := func(doc document) error {
		changed++
		if preferFullRebuild(indexedBefore, changed) {
			return errIncrementalTooLarge
		}
		upserts = append(upserts, doc)
		if len(upserts) >= indexBatchSize {
			return flushUpserts()
		}
		return nil
	}
	queueDelete := func(key string) error {
		changed++
		if preferFullRebuild(indexedBefore, changed) {
			return errIncrementalTooLarge
		}
		deletes = append(deletes, key)
		if len(deletes) >= indexBatchSize {
			return flushDeletes()
		}
		return nil
	}

	if err := e.deltaMediaFiles(ctx, ds, since, queueUpsert, queueDelete); err != nil {
		if errors.Is(err, errIncrementalTooLarge) {
			return e.rebuildLocked(ctx, ds, libraries)
		}
		return err
	}
	if err := e.deltaAlbums(ctx, ds, since, queueUpsert, queueDelete); err != nil {
		if errors.Is(err, errIncrementalTooLarge) {
			return e.rebuildLocked(ctx, ds, libraries)
		}
		return err
	}
	if err := e.deltaArtists(ctx, ds, libraries, since, queueUpsert, queueDelete); err != nil {
		if errors.Is(err, errIncrementalTooLarge) {
			return e.rebuildLocked(ctx, ds, libraries)
		}
		return err
	}
	if err := flushUpserts(); err != nil {
		return err
	}
	if err := flushDeletes(); err != nil {
		return err
	}
	if changed > 0 {
		resp, err := e.roundTrip(ctx, request{Op: "commit"})
		if err != nil {
			return err
		}
		lastIndexed = resp.Indexed
	}
	if changed == 0 {
		if indexedBefore != expected {
			log.Debug(ctx, "Rust search document count drifted without deltas; rebuilding",
				"indexed", indexedBefore, "expected", expected)
			return e.rebuildLocked(ctx, ds, libraries)
		}
		e.generation.Store(scanGeneration(libraries))
		return nil
	}
	if int64(lastIndexed) != expected {
		log.Debug(ctx, "Rust search document count drifted after incremental refresh; rebuilding",
			"indexed", lastIndexed, "expected", expected)
		return e.rebuildLocked(ctx, ds, libraries)
	}

	e.indexed.Store(lastIndexed)
	e.generation.Store(scanGeneration(libraries))
	e.ready.Store(true)
	log.Info(ctx, "Rust search index refreshed incrementally", "documents", lastIndexed, "changed", changed)
	return nil
}

var errIncrementalTooLarge = errors.New("rust search incremental delta is too large")

func preferFullRebuild(indexed, delta int64) bool {
	if indexed <= 0 || delta <= 0 {
		return false
	}
	return float64(delta) > float64(indexed)*maxIncrementalRatio
}

func abs64(v int64) int64 {
	if v < 0 {
		return -v
	}
	return v
}

func expectedSearchDocuments(ctx context.Context, ds model.DataStore) (int64, error) {
	songs, err := ds.MediaFile(ctx).CountAll(model.QueryOptions{Filters: query.NotMissing()})
	if err != nil {
		return 0, fmt.Errorf("counting media files for Rust search: %w", err)
	}
	albums, err := ds.Album(ctx).CountAll(model.QueryOptions{Filters: query.NotMissing()})
	if err != nil {
		return 0, fmt.Errorf("counting albums for Rust search: %w", err)
	}
	artists, err := ds.Artist(ctx).CountAll(model.QueryOptions{Filters: query.NotMissing()})
	if err != nil {
		return 0, fmt.Errorf("counting artists for Rust search: %w", err)
	}
	return songs + albums + artists, nil
}

func (e *Engine) Rebuild(ctx context.Context, ds model.DataStore) error {
	if e.skipBackgroundWork() {
		return nil
	}
	if !e.building.CompareAndSwap(false, true) {
		return nil
	}
	defer e.building.Store(false)

	ctx = auth.WithAdminUser(ctx, ds)
	libraries, err := ds.Library(ctx).GetAll()
	if err != nil {
		return fmt.Errorf("loading libraries for Rust search: %w", err)
	}
	return e.rebuildLocked(ctx, ds, libraries)
}

func (e *Engine) rebuildLocked(ctx context.Context, ds model.DataStore, libraries model.Libraries) error {
	wasReady := e.ready.Load()
	if _, err := e.roundTrip(ctx, request{Op: "begin_replace"}); err != nil {
		return err
	}
	committed := false
	defer func() {
		if !committed {
			cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), time.Second)
			defer cancel()
			if _, abortErr := e.roundTrip(cleanupCtx, request{Op: "abort_replace"}); abortErr == nil && wasReady {
				e.ready.Store(true)
			}
		}
	}()

	batch := make([]document, 0, indexBatchSize)
	flush := func() error {
		if len(batch) == 0 {
			return nil
		}
		_, err := e.roundTrip(ctx, request{Op: "append", Documents: batch})
		batch = batch[:0]
		return err
	}
	appendDocument := func(doc document) error {
		batch = append(batch, doc)
		if len(batch) >= indexBatchSize {
			return flush()
		}
		return nil
	}

	if err := e.indexMediaFiles(ctx, ds, appendDocument); err != nil {
		return err
	}
	if err := e.indexAlbums(ctx, ds, appendDocument); err != nil {
		return err
	}
	if err := e.indexArtists(ctx, ds, libraries, appendDocument); err != nil {
		return err
	}
	if err := flush(); err != nil {
		return err
	}
	resp, err := e.roundTrip(ctx, request{Op: "commit_replace"})
	if err != nil {
		return err
	}
	committed = true
	e.indexed.Store(resp.Indexed)
	e.generation.Store(scanGeneration(libraries))
	e.ready.Store(true)
	log.Info(ctx, "Rust search index ready", "documents", resp.Indexed)
	return nil
}

func (e *Engine) indexMediaFiles(ctx context.Context, ds model.DataStore, appendDocument func(document) error) error {
	cursor, err := ds.MediaFile(ctx).GetCursor()
	if err != nil {
		return fmt.Errorf("opening media file cursor for Rust search: %w", err)
	}
	for mediaFile, cursorErr := range cursor {
		if cursorErr != nil {
			return fmt.Errorf("reading media files for Rust search: %w", cursorErr)
		}
		if mediaFile.Missing {
			continue
		}
		// FTS secondary variants are applied in navidrome-search Apply (fts-normalize).
		if err := appendDocument(e.mediaFileDocument(ctx, mediaFile)); err != nil {
			return err
		}
	}
	return nil
}

func (e *Engine) deltaMediaFiles(ctx context.Context, ds model.DataStore, since time.Time, upsert func(document) error, deleteKey func(string) error) error {
	cursor, err := ds.MediaFile(ctx).GetCursor(model.QueryOptions{Filters: query.Or(
		query.ColumnAfter("media_file.created_at", since),
		query.ColumnAfter("media_file.updated_at", since),
	)})
	if err != nil {
		return fmt.Errorf("opening media file delta cursor for Rust search: %w", err)
	}
	for mediaFile, cursorErr := range cursor {
		if cursorErr != nil {
			return fmt.Errorf("reading media file deltas for Rust search: %w", cursorErr)
		}
		if mediaFile.Missing {
			if err := deleteKey("song:" + mediaFile.ID); err != nil {
				return err
			}
			continue
		}
		if err := upsert(e.mediaFileDocument(ctx, mediaFile)); err != nil {
			return err
		}
	}
	return nil
}

func (e *Engine) mediaFileDocument(_ context.Context, mediaFile model.MediaFile) document {
	secondary := []string{mediaFile.Album, mediaFile.Artist, mediaFile.AlbumArtist,
		mediaFile.SortTitle, mediaFile.SortAlbumName, mediaFile.SortArtistName, mediaFile.SortAlbumArtistName}
	secondary = append(secondary, mediaFile.Participants.AllNames()...)
	// Prefer DB search_normalized when present (ingest path). Missing values are
	// filled in-process by navidrome-search Apply via fts-normalize — no metadata hop.
	if norm := mediaFile.SearchNormalized; norm != "" {
		secondary = append(secondary, norm)
	}
	return document{
		Key: "song:" + mediaFile.ID, ID: mediaFile.ID, Kind: "song",
		LibraryIDs: []uint64{uint64(mediaFile.LibraryID)}, Primary: mediaFile.FullTitle(),
		Secondary: strings.Join(secondary, " "),
	}
}

func (e *Engine) indexAlbums(ctx context.Context, ds model.DataStore, appendDocument func(document) error) error {
	cursor, err := ds.Album(ctx).GetCursor()
	if err != nil {
		return fmt.Errorf("opening album cursor for Rust search: %w", err)
	}
	for album, cursorErr := range cursor {
		if cursorErr != nil {
			return fmt.Errorf("reading albums for Rust search: %w", cursorErr)
		}
		if album.Missing {
			continue
		}
		if err := appendDocument(e.albumDocument(ctx, album)); err != nil {
			return err
		}
	}
	return nil
}

func (e *Engine) deltaAlbums(ctx context.Context, ds model.DataStore, since time.Time, upsert func(document) error, deleteKey func(string) error) error {
	cursor, err := ds.Album(ctx).GetCursor(model.QueryOptions{Filters: query.Or(
		query.ColumnAfter("album.created_at", since),
		query.ColumnAfter("album.updated_at", since),
		query.ColumnAfter("album.imported_at", since),
	)})
	if err != nil {
		return fmt.Errorf("opening album delta cursor for Rust search: %w", err)
	}
	for album, cursorErr := range cursor {
		if cursorErr != nil {
			return fmt.Errorf("reading album deltas for Rust search: %w", cursorErr)
		}
		if album.Missing {
			if err := deleteKey("album:" + album.ID); err != nil {
				return err
			}
			continue
		}
		if err := upsert(e.albumDocument(ctx, album)); err != nil {
			return err
		}
	}
	return nil
}

func (e *Engine) albumDocument(_ context.Context, album model.Album) document {
	secondary := []string{album.AlbumArtist, album.SortAlbumName, album.SortAlbumArtistName,
		album.CatalogNum, strings.Join(album.Participants.AllNames(), " ")}
	if norm := album.SearchNormalized; norm != "" {
		secondary = append(secondary, norm)
	}
	return document{
		Key: "album:" + album.ID, ID: album.ID, Kind: "album",
		LibraryIDs: []uint64{uint64(album.LibraryID)}, Primary: album.FullName(),
		Secondary: strings.Join(secondary, " "),
	}
}

func (e *Engine) indexArtists(ctx context.Context, ds model.DataStore, libraries model.Libraries, appendDocument func(document) error) error {
	return e.collectArtists(ctx, ds, libraries, nil, func(doc document, missing bool) error {
		if missing {
			return nil
		}
		return appendDocument(doc)
	})
}

func (e *Engine) deltaArtists(ctx context.Context, ds model.DataStore, libraries model.Libraries, since time.Time, upsert func(document) error, deleteKey func(string) error) error {
	return e.collectArtists(ctx, ds, libraries, query.ColumnAfter("artist.updated_at", since), func(doc document, missing bool) error {
		if missing {
			return deleteKey(doc.Key)
		}
		return upsert(doc)
	})
}

func (e *Engine) collectArtists(ctx context.Context, ds model.DataStore, libraries model.Libraries, extraFilter query.Sqlizer, emit func(document, bool) error) error {
	if len(libraries) == 0 {
		return nil
	}
	libraryIDs := libraries.IDs()
	requested := make(map[int]struct{}, len(libraryIDs))
	for _, id := range libraryIDs {
		requested[id] = struct{}{}
	}
	// One GetAll for all libraries; LibraryIDs recovered from library_stats_json.
	artists, err := ds.Artist(ctx).GetAll(model.QueryOptions{Filters: query.And(
		query.Eq("library_id", libraryIDs),
		extraFilter,
	)})
	if err != nil {
		return fmt.Errorf("loading artists for Rust search: %w", err)
	}
	for _, artist := range artists {
		ids := make([]uint64, 0, len(artist.LibraryIDs))
		for _, id := range artist.LibraryIDs {
			if _, ok := requested[id]; ok {
				ids = append(ids, uint64(id))
			}
		}
		if len(ids) == 0 {
			// Defensive: filtered join should always populate LibraryIDs.
			continue
		}
		doc := e.artistDocument(ctx, artist, ids)
		if err := emit(doc, artist.Missing); err != nil {
			return err
		}
	}
	return nil
}

func (e *Engine) artistDocument(_ context.Context, artist model.Artist, libraryIDs []uint64) document {
	secondary := []string{artist.SortArtistName, artist.OrderArtistName}
	if norm := artist.SearchNormalized; norm != "" {
		secondary = append(secondary, norm)
	}
	return document{
		Key: "artist:" + artist.ID, ID: artist.ID, Kind: "artist",
		LibraryIDs: libraryIDs, Primary: artist.Name,
		Secondary: strings.Join(secondary, " "),
	}
}

func scanGeneration(libraries model.Libraries) int64 {
	var latest int64
	for _, library := range libraries {
		latest = max(latest, library.LastScanAt.UnixNano())
		latest = max(latest, library.UpdatedAt.UnixNano())
	}
	return latest
}

func searchIndexStale(ready bool, generation int64, libraries model.Libraries) bool {
	return !ready || scanGeneration(libraries) > generation
}

func (e *Engine) roundTrip(ctx context.Context, req request) (response, error) {
	timeout := searchRequestTimeout
	switch req.Op {
	case "begin_replace", "append", "commit_replace", "abort_replace", "upsert", "delete", "commit":
		timeout = indexRequestTimeout
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	if err := ctx.Err(); err != nil {
		return response{}, err
	}
	e.gate.Lock()
	if err := e.ensureWorker(); err != nil {
		e.gate.Unlock()
		return response{}, err
	}
	if e.grpc == nil {
		e.gate.Unlock()
		return response{}, fmt.Errorf("search gRPC worker not started")
	}
	// gRPC is multiplexed; do not hold the stdin gate across the RPC.
	e.gate.Unlock()
	resp, err := e.grpcRoundTrip(ctx, req)
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) || ctx.Err() != nil {
			return response{}, err
		}
		if rustworker.IsTransportFailure(err) {
			e.gate.Lock()
			// Drop the client so the next call redials. Keep ready=true: the
			// on-disk Tantivy index survives a worker restart.
			e.closeGRPC()
			e.gate.Unlock()
		}
		return response{}, err
	}
	return resp, nil
}

func (e *Engine) ensureWorker() error {
	if e.grpc != nil {
		return nil
	}
	if err := e.startGRPC(); err != nil {
		return fmt.Errorf("search gRPC worker unavailable: %w", err)
	}
	return nil
}

func (e *Engine) stopWorker() {
	e.ready.Store(false)
	e.indexed.Store(0)
	e.closeGRPC()
}

func (e *Engine) closeGRPC() {
	searchworker.InvalidateGRPC()
	e.grpc = nil
}
