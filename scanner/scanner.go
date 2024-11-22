package scanner

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"

	ppl "github.com/google/go-pipeline/pkg/pipeline"
	"github.com/navidrome/navidrome/core/artwork"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/utils/chain"
)

type scannerImpl struct {
	ds model.DataStore
	cw artwork.CacheWarmer
}

func (s *scannerImpl) scanAll(ctx context.Context, fullRescan bool, progress chan<- *scannerStatus) {
	libs, err := s.ds.Library(ctx).GetAll()
	if err != nil {
		progress <- &scannerStatus{err: fmt.Errorf("failed to get libraries: %w", err)}
		return
	}

	startTime := time.Now()
	log.Info(ctx, "Scanner: Starting scan", "fullRescan", fullRescan, "numLibraries", len(libs))
	changesDetected := atomic.Bool{}

	err = chain.RunSequentially(
		// Phase 1: Scan all libraries and import new/updated files
		func() error {
			return runPhase[*folderEntry](ctx, 1, createPhaseFolders(ctx, s.ds, s.cw, libs, fullRescan, &changesDetected, progress))
		},

		// Phase 2: Process missing files, checking for moves
		func() error { return runPhase[*missingTracks](ctx, 2, createPhaseMissingTracks(ctx, s.ds)) },

		// Phase 3: Refresh all new/changed albums and update artists
		func() error { return runPhase[*model.Album](ctx, 3, createPhaseRefreshAlbums(ctx, s.ds, libs)) },
	)
	if err != nil {
		log.Error(ctx, "Scanner: Finished with error", "duration", time.Since(startTime), err)
		progress <- &scannerStatus{err: err}
		return
	}

	// Run GC if there were any changes (Remove dangling tracks, empty albums and artists, and orphan annotations)
	if changesDetected.Load() {
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
				log.Error(ctx, "Scanner: Error updating last scan completed at", "lib", lib.Name, err)
			}
		}
		return nil
	})

	log.Info(ctx, "Scanner: Finished scanning all libraries", "duration", time.Since(startTime))
}

type phase[T any] interface {
	producer() ppl.Producer[T]
	stages() []ppl.Stage[T]
	finalize(error) error
	description() string
}

func runPhase[T any](ctx context.Context, phaseNum int, phase phase[T]) error {
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

func countTasks[T any]() (*atomic.Int64, func(T) (T, error)) {
	counter := atomic.Int64{}
	return &counter, func(in T) (T, error) {
		counter.Add(1)
		return in, nil
	}
}

func (s *scannerImpl) Status(context.Context) (*StatusInfo, error) {
	return &StatusInfo{}, nil
}

var _ scanner = (*scannerImpl)(nil)
