package scanner

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"

	ppl "github.com/google/go-pipeline/pkg/pipeline"
	"github.com/navidrome/navidrome/core"
	"github.com/navidrome/navidrome/core/artwork"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/utils/chain"
)

type scannerImpl struct {
	ds  model.DataStore
	cw  artwork.CacheWarmer
	pls core.Playlists
}

// scanState holds the state of a in-progress scan, to be passed to the various phases
type scanState struct {
	progress        chan<- *ProgressInfo
	fullScan        bool
	changesDetected atomic.Bool
}

func (s *scanState) sendProgress(info *ProgressInfo) {
	if s.progress != nil {
		s.progress <- info
	}
}

func (s *scanState) sendError(err error) {
	s.sendProgress(&ProgressInfo{Err: err})
}

func (s *scannerImpl) scanAll(ctx context.Context, fullScan bool, progress chan<- *ProgressInfo) {
	state := scanState{progress: progress, fullScan: fullScan}
	libs, err := s.ds.Library(ctx).GetAll()
	if err != nil {
		state.sendProgress(&ProgressInfo{Err: fmt.Errorf("getting libraries: %w", err)})
		return
	}

	startTime := time.Now()
	log.Info(ctx, "Scanner: Starting scan", "fullScan", fullScan, "numLibraries", len(libs))

	err = chain.RunSequentially(
		// Phase 1: Scan all libraries and import new/updated files
		runPhase[*folderEntry](ctx, 1, createPhaseFolders(ctx, &state, s.ds, s.cw, libs)),

		// Phase 2: Process missing files, checking for moves
		runPhase[*missingTracks](ctx, 2, createPhaseMissingTracks(ctx, &state, s.ds)),

		chain.RunParallel(
			// Phase 3: Refresh all new/changed albums and update artists
			runPhase[*model.Album](ctx, 3, createPhaseRefreshAlbums(ctx, &state, s.ds, libs)),

			// Phase 4: Import/update playlists
			runPhase[*model.Folder](ctx, 4, createPhasePlaylists(ctx, &state, s.ds, s.pls, s.cw)),
		),
	)
	if err != nil {
		log.Error(ctx, "Scanner: Finished with error", "duration", time.Since(startTime), err)
		state.sendError(err)
		return
	}

	// Run GC if there were any changes (Remove dangling tracks, empty albums and artists, and orphan annotations)
	if state.changesDetected.Load() {
		_ = s.ds.WithTx(func(tx model.DataStore) error {
			start := time.Now()
			err := tx.GC(ctx)
			if err != nil {
				log.Error(ctx, "Scanner: Error running GC", err)
				return err
			}
			log.Debug(ctx, "Scanner: GC completed", "duration", time.Since(start))
			return nil
		})
	} else {
		log.Debug(ctx, "Scanner: No changes detected, skipping GC")
	}

	// Final step: Update last_scan_completed_at for all libraries
	_ = s.ds.WithTx(func(tx model.DataStore) error {
		for _, lib := range libs {
			err := tx.Library(ctx).UpdateLastScanCompletedAt(lib.ID, time.Now())
			if err != nil {
				state.sendProgress(&ProgressInfo{Err: fmt.Errorf("updating last scan completed: %w", err)})
				log.Error(ctx, "Scanner: Error updating last scan completed", "lib", lib.Name, err)
			}
		}
		return nil
	})

	if state.changesDetected.Load() {
		state.sendProgress(&ProgressInfo{ChangesDetected: true})
	}

	log.Info(ctx, "Scanner: Finished scanning all libraries", "duration", time.Since(startTime))
}

type phase[T any] interface {
	producer() ppl.Producer[T]
	stages() []ppl.Stage[T]
	finalize(error) error
	description() string
}

func runPhase[T any](ctx context.Context, phaseNum int, phase phase[T]) func() error {
	return func() error {
		log.Debug(ctx, fmt.Sprintf("Scanner: Starting phase %d: %s", phaseNum, phase.description()))
		start := time.Now()

		producer := phase.producer()
		stages := phase.stages()

		// Prepend a counter stage to the phase's pipeline
		counter, countStageFn := countTasks[T]()
		stages = append([]ppl.Stage[T]{ppl.NewStage(countStageFn, ppl.Name("count tasks"))}, stages...)

		var err error
		if log.IsGreaterOrEqualTo(log.LevelDebug) {
			var metrics *ppl.Metrics
			metrics, err = ppl.Measure(producer, stages...)
			log.Info(ctx, "Scanner: "+metrics.String(), err)
		} else {
			err = ppl.Do(producer, stages...)
		}

		err = phase.finalize(err)

		if err != nil {
			log.Error(ctx, fmt.Sprintf("Scanner: Error processing libraries in phase %d", phaseNum), "elapsed", time.Since(start), err)
		} else {
			log.Debug(ctx, fmt.Sprintf("Scanner: Finished phase %d", phaseNum), "elapsed", time.Since(start), "totalTasks", counter.Load())
		}

		return err
	}
}

func countTasks[T any]() (*atomic.Int64, func(T) (T, error)) {
	counter := atomic.Int64{}
	return &counter, func(in T) (T, error) {
		counter.Add(1)
		return in, nil
	}
}

var _ scanner = (*scannerImpl)(nil)
