package scanner2

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"

	ppl "github.com/google/go-pipeline/pkg/pipeline"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
	"github.com/navidrome/navidrome/scanner"
	"github.com/navidrome/navidrome/utils/singleton"
)

type scanner2 struct {
	rootCtx context.Context
	ds      model.DataStore
}

func GetInstance(rootCtx context.Context, ds model.DataStore) scanner.Scanner {
	return singleton.GetInstance(func() *scanner2 {
		return &scanner2{rootCtx: rootCtx, ds: ds}
	})
}

func (s *scanner2) RescanAll(requestCtx context.Context, fullRescan bool) error {
	ctx := request.AddValues(s.rootCtx, requestCtx)

	libs, err := s.ds.Library(ctx).GetAll()
	if err != nil {
		return err
	}

	startTime := time.Now()
	log.Info(ctx, "Scanner: Starting scan", "fullRescan", fullRescan, "numLibraries", len(libs))

	// Phase 1: Scan all libraries and import new/updated files
	err = runPipeline(ctx, 1,
		ppl.NewProducer(produceFolders(ctx, s.ds, libs, fullRescan), ppl.Name("read folders from disk")),
		ppl.NewStage(processFolder(ctx), ppl.Name("process folder")),
		ppl.NewStage(persistChanges(ctx), ppl.Name("persist changes")),
		ppl.NewStage(logFolder(ctx), ppl.Name("log results")),
	)

	// Phase 2: Process missing files, checking for moves
	if err == nil {
		err = runPipeline(ctx, 2,
			ppl.NewProducer(produceMissingTracks(ctx, s.ds), ppl.Name("load missing tracks from db")),
			ppl.NewStage(processMissingTracks(ctx, s.ds), ppl.Name("detect moved songs")),
		)
	}

	if err == nil {
		err = runPipeline(ctx, 3,
			ppl.NewProducer(produceOutdatedAlbums(ctx, s.ds, libs), ppl.Name("load albums from db")),
			ppl.NewStage(refreshAlbums(ctx, s.ds), ppl.Name("refresh albums")),
		)
	}

	if err != nil {
		log.Error(ctx, "Scanner: Finished with error", "duration", time.Since(startTime), err)
		return err
	}

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
	return nil
}

func runPipeline[T any](ctx context.Context, phase int, producer ppl.Producer[T], stages ...ppl.Stage[T]) error {
	log.Debug(ctx, fmt.Sprintf("Scanner: Starting phase %d", phase))
	start := time.Now()

	counter, countStageFn := countTasks[T]()
	stages = append(stages, ppl.NewStage(countStageFn, ppl.Name("count tasks")))

	var err error
	if log.IsGreaterOrEqualTo(log.LevelDebug) {
		var metrics *ppl.Metrics
		metrics, err = ppl.Measure(producer, stages...)
		log.Info(metrics.String(), err)
	} else {
		err = ppl.Do(producer, stages...)
	}
	if err != nil {
		log.Error(ctx, fmt.Sprintf("Scanner: Error processing libraries in phase %d", phase), "elapsed", time.Since(start), err)
	} else {
		log.Debug(ctx, fmt.Sprintf("Scanner: Finished phase %d", phase), "elapsed", time.Since(start), "totalTasks", counter.Load())
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

func logFolder(ctx context.Context) func(entry *folderEntry) (out *folderEntry, err error) {
	return func(entry *folderEntry) (*folderEntry, error) {
		log.Debug(ctx, "Scanner: Completed processing folder", " path", entry.path,
			"audioCount", len(entry.audioFiles), "imageCount", len(entry.imageFiles), "plsCount", len(entry.playlists),
			"elapsed", time.Since(entry.startTime))
		return entry, nil
	}
}

func (s *scanner2) Status(context.Context) (*scanner.StatusInfo, error) {
	return &scanner.StatusInfo{}, nil
}

var _ scanner.Scanner = (*scanner2)(nil)
