package playlists

import (
	"context"
	"maps"
	"slices"
	"sync"
	"time"

	"github.com/navidrome/navidrome/core/auth"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
)

// SmartPlaylistEvaluator evaluates smart playlists in the background.
// Call Enqueue to queue a playlist for evaluation. The evaluation happens
// asynchronously in a background goroutine.
type SmartPlaylistEvaluator interface {
	Enqueue(playlistID string)
}

func NewSmartPlaylistEvaluator(ds model.DataStore) SmartPlaylistEvaluator {
	e := &smartPlaylistEvaluator{
		ds:         ds,
		buffer:     make(map[string]struct{}),
		wakeSignal: make(chan struct{}, 1),
	}
	go e.run()
	return e
}

type smartPlaylistEvaluator struct {
	ds         model.DataStore
	buffer     map[string]struct{}
	mutex      sync.Mutex
	wakeSignal chan struct{}
}

func (e *smartPlaylistEvaluator) Enqueue(playlistID string) {
	e.mutex.Lock()
	defer e.mutex.Unlock()
	e.buffer[playlistID] = struct{}{}
	e.sendWakeSignal()
}

func (e *smartPlaylistEvaluator) sendWakeSignal() {
	select {
	case e.wakeSignal <- struct{}{}:
	default:
	}
}

func (e *smartPlaylistEvaluator) run() {
	for {
		e.waitSignal(10 * time.Second)

		e.mutex.Lock()
		if len(e.buffer) == 0 {
			e.mutex.Unlock()
			continue
		}

		batch := slices.Collect(maps.Keys(e.buffer))
		e.buffer = make(map[string]struct{})
		e.mutex.Unlock()

		e.processBatch(batch)
	}
}

func (e *smartPlaylistEvaluator) waitSignal(timeout time.Duration) {
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	select {
	case <-timer.C:
	case <-e.wakeSignal:
	}
}

func (e *smartPlaylistEvaluator) processBatch(batch []string) {
	// Use admin context so userFilter() returns all playlists.
	// Evaluate() internally uses pls.OwnerID for annotation JOINs.
	ctx := auth.WithAdminUser(context.TODO(), e.ds)

	log.Debug(ctx, "Evaluating smart playlists in background", "count", len(batch))
	for _, id := range batch {
		start := time.Now()
		err := e.ds.Playlist(ctx).Evaluate(id)
		if err != nil {
			log.Error(ctx, "Error evaluating smart playlist in background", "id", id, err)
			continue
		}
		log.Debug(ctx, "Smart playlist evaluation complete", "id", id, "elapsed", time.Since(start))
	}
}

// NoopSmartPlaylistEvaluator returns an evaluator that does nothing.
// Used in CLI scan and test contexts.
func NoopSmartPlaylistEvaluator() SmartPlaylistEvaluator {
	return &noopSmartPlaylistEvaluator{}
}

type noopSmartPlaylistEvaluator struct{}

func (n *noopSmartPlaylistEvaluator) Enqueue(string) {}
