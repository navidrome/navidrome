package scanner

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"

	ppl "github.com/google/go-pipeline/pkg/pipeline"
	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/core"
	"github.com/navidrome/navidrome/core/artwork"
	"github.com/navidrome/navidrome/db"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/utils/run"
)

type scannerImpl struct {
	ds  model.DataStore
	cw  artwork.CacheWarmer
	pls core.Playlists
}

// scanState holds the state of an in-progress scan, to be passed to the various phases
type scanState struct {
	progress        chan<- *ProgressInfo
	fullScan        bool
	changesDetected atomic.Bool
	libraries       model.Libraries // Store libraries list for consistency across phases
}

func (s *scanState) sendProgress(info *ProgressInfo) {
	if s.progress != nil {
		s.progress <- info
	}
}

func (s *scanState) sendWarning(msg string) {
	s.sendProgress(&ProgressInfo{Warning: msg})
}

func (s *scanState) sendError(err error) {
	s.sendProgress(&ProgressInfo{Error: err.Error()})
}

func (s *scannerImpl) scanAll(ctx context.Context, fullScan bool, progress chan<- *ProgressInfo) {
	startTime := time.Now()

	state := scanState{
		progress:        progress,
		fullScan:        fullScan,
		changesDetected: atomic.Bool{},
	}

	// Set changesDetected to true for full scans to ensure all maintenance operations run
	if fullScan {
		state.changesDetected.Store(true)
	}

	libs, err := s.ds.Library(ctx).GetAll()
	if err != nil {
		state.sendWarning(fmt.Sprintf("getting libraries: %s", err))
		return
	}
	state.libraries = libs

	log.Info(ctx, "Scanner: Starting scan", "fullScan", state.fullScan, "numLibraries", len(libs))

	// Store scan type and start time
	scanType := "quick"
	if state.fullScan {
		scanType = "full"
	}
	_ = s.ds.Property(ctx).Put(consts.LastScanTypeKey, scanType)
	_ = s.ds.Property(ctx).Put(consts.LastScanStartTimeKey, startTime.Format(time.RFC3339))

	// if there was a full scan in progress, force a full scan
	if !state.fullScan {
		for _, lib := range libs {
			if lib.FullScanInProgress {
				log.Info(ctx, "Scanner: Interrupted full scan detected", "lib", lib.Name)
				state.fullScan = true
				_ = s.ds.Property(ctx).Put(consts.LastScanTypeKey, "full")
				break
			}
		}
	}

	err = run.Sequentially(
		// Phase 1: Scan all libraries and import new/updated files
		runPhase[*folderEntry](ctx, 1, createPhaseFolders(ctx, &state, s.ds, s.cw, libs)),

		// Phase 2: Process missing files, checking for moves
		runPhase[*missingTracks](ctx, 2, createPhaseMissingTracks(ctx, &state, s.ds)),

		// Phases 3 and 4 can be run in parallel
		run.Parallel(
			// Phase 3: Refresh all new/changed albums and update artists
			runPhase[*model.Album](ctx, 3, createPhaseRefreshAlbums(ctx, &state, s.ds, libs)),

			// Phase 4: Import/update playlists
			runPhase[*model.Folder](ctx, 4, createPhasePlaylists(ctx, &state, s.ds, s.pls, s.cw)),
		),

		// Final Steps (cannot be parallelized):

		// Run GC if there were any changes (Remove dangling tracks, empty albums and artists, and orphan annotations)
		s.runGC(ctx, &state),

		// Refresh artist and tags stats
		s.runRefreshStats(ctx, &state),

		// Update last_scan_completed_at for all libraries
		s.runUpdateLibraries(ctx, &state),

		// Optimize DB
		s.runOptimize(ctx),
	)
	if err != nil {
		log.Error(ctx, "Scanner: Finished with error", "duration", time.Since(startTime), err)
		_ = s.ds.Property(ctx).Put(consts.LastScanErrorKey, err.Error())
		state.sendError(err)
		return
	}

	_ = s.ds.Property(ctx).Put(consts.LastScanErrorKey, "")

	if state.changesDetected.Load() {
		state.sendProgress(&ProgressInfo{ChangesDetected: true})
	}

	log.Info(ctx, "Scanner: Finished scanning all libraries", "duration", time.Since(startTime))
}

func (s *scannerImpl) runGC(ctx context.Context, state *scanState) func() error {
	return func() error {
		state.sendProgress(&ProgressInfo{ForceUpdate: true})
		return s.ds.WithTx(func(tx model.DataStore) error {
			if state.changesDetected.Load() {
				start := time.Now()
				err := tx.GC(ctx)
				if err != nil {
					log.Error(ctx, "Scanner: Error running GC", err)
					return fmt.Errorf("running GC: %w", err)
				}
				log.Debug(ctx, "Scanner: GC completed", "elapsed", time.Since(start))
			} else {
				log.Debug(ctx, "Scanner: No changes detected, skipping GC")
			}
			return nil
		}, "scanner: GC")
	}
}

func (s *scannerImpl) runRefreshStats(ctx context.Context, state *scanState) func() error {
	return func() error {
		if !state.changesDetected.Load() {
			log.Debug(ctx, "Scanner: No changes detected, skipping refreshing stats")
			return nil
		}
		start := time.Now()
		stats, err := s.ds.Artist(ctx).RefreshStats(state.fullScan)
		if err != nil {
			log.Error(ctx, "Scanner: Error refreshing artists stats", err)
			return fmt.Errorf("refreshing artists stats: %w", err)
		}
		log.Debug(ctx, "Scanner: Refreshed artist stats", "stats", stats, "elapsed", time.Since(start))

		start = time.Now()
		err = s.ds.Tag(ctx).UpdateCounts()
		if err != nil {
			log.Error(ctx, "Scanner: Error updating tag counts", err)
			return fmt.Errorf("updating tag counts: %w", err)
		}
		log.Debug(ctx, "Scanner: Updated tag counts", "elapsed", time.Since(start))
		return nil
	}
}

func (s *scannerImpl) runOptimize(ctx context.Context) func() error {
	return func() error {
		start := time.Now()
		db.Optimize(ctx)
		log.Debug(ctx, "Scanner: Optimized DB", "elapsed", time.Since(start))
		return nil
	}
}

func (s *scannerImpl) runUpdateLibraries(ctx context.Context, state *scanState) func() error {
	return func() error {
		start := time.Now()
		return s.ds.WithTx(func(tx model.DataStore) error {
			for _, lib := range state.libraries {
				err := tx.Library(ctx).ScanEnd(lib.ID)
				if err != nil {
					log.Error(ctx, "Scanner: Error updating last scan completed", "lib", lib.Name, err)
					return fmt.Errorf("updating last scan completed: %w", err)
				}
				err = tx.Property(ctx).Put(consts.PIDTrackKey, conf.Server.PID.Track)
				if err != nil {
					log.Error(ctx, "Scanner: Error updating track PID conf", err)
					return fmt.Errorf("updating track PID conf: %w", err)
				}
				err = tx.Property(ctx).Put(consts.PIDAlbumKey, conf.Server.PID.Album)
				if err != nil {
					log.Error(ctx, "Scanner: Error updating album PID conf", err)
					return fmt.Errorf("updating album PID conf: %w", err)
				}
				if state.changesDetected.Load() {
					log.Debug(ctx, "Scanner: Refreshing library stats", "lib", lib.Name)
					if err := tx.Library(ctx).RefreshStats(lib.ID); err != nil {
						log.Error(ctx, "Scanner: Error refreshing library stats", "lib", lib.Name, err)
						return fmt.Errorf("refreshing library stats: %w", err)
					}
				} else {
					log.Debug(ctx, "Scanner: No changes detected, skipping library stats refresh", "lib", lib.Name)
				}
			}
			log.Debug(ctx, "Scanner: Updated libraries after scan", "elapsed", time.Since(start), "numLibraries", len(state.libraries))
			return nil
		}, "scanner: update libraries")
	}
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
			var m *ppl.Metrics
			m, err = ppl.Measure(producer, stages...)
			log.Info(ctx, "Scanner: "+m.String(), err)
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
