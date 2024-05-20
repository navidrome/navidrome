package scanner2

import (
	"context"
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

	err = s.runPipeline(
		ppl.NewProducer(produceFolders(ctx, s.ds, libs, fullRescan), ppl.Name("read folders from disk")),
		ppl.NewStage(logFolder(ctx), ppl.Name("log results")),
	)

	if err != nil {
		log.Error(ctx, "Scanner: Error scanning libraries", "duration", time.Since(startTime), err)
	} else {
		log.Info(ctx, "Scanner: Finished scanning all libraries", "duration", time.Since(startTime))
	}
	return err
}

func (s *scanner2) runPipeline(producer ppl.Producer[*folderEntry], stages ...ppl.Stage[*folderEntry]) error {
	if log.IsGreaterOrEqualTo(log.LevelDebug) {
		metrics, err := ppl.Measure(producer, stages...)
		log.Info(metrics.String(), err)
		return err
	}
	return ppl.Do(producer, stages...)
}

func logFolder(ctx context.Context) func(folder *folderEntry) (out *folderEntry, err error) {
	return func(folder *folderEntry) (out *folderEntry, err error) {
		log.Debug(ctx, "Scanner: Completed processing folder", " path", folder.path,
			"audioCount", len(folder.audioFiles), "imageCount", len(folder.imageFiles), "plsCount", len(folder.playlists),
			"elapsed", time.Since(folder.startTime))
		return folder, nil
	}
}

func (s *scanner2) Status(requestCtx context.Context) (*scanner.StatusInfo, error) {
	return &scanner.StatusInfo{}, nil
}

var _ scanner.Scanner = (*scanner2)(nil)
