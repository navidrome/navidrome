package scanner2

import (
	"context"

	ppl "github.com/google/go-pipeline/pkg/pipeline"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
)

type missingTracks struct {
	lib     model.Library
	pid     string
	missing model.MediaFiles
	matched model.MediaFiles
}

func produceMissingTracks(ctx context.Context, ds model.DataStore) ppl.ProducerFn[*missingTracks] {
	return func(put func(tracks *missingTracks)) error {
		libs, err := ds.Library(ctx).GetAll()
		if err != nil {
			return err
		}
		for _, lib := range libs {
			if lib.LastScanStartedAt.IsZero() {
				continue
			}
			mfs, err := ds.MediaFile(ctx).GetMissingAndMatching(lib.ID)
			if err != nil {
				return err
			}
			mt := missingTracks{lib: lib}
			for _, mf := range mfs {
				if mt.pid != mf.PID {
					if mt.pid != "" {
						put(&mt)
					}
					mt = missingTracks{lib: lib}
				}
				mt.pid = mf.PID
				if mf.Missing {
					mt.missing = append(mt.missing, mf)
				} else {
					mt.matched = append(mt.matched, mf)
				}
			}
			if mt.pid != "" {
				put(&mt)
			}
		}

		return nil
	}
}

func processMissingTracks(ctx context.Context, ds model.DataStore) ppl.StageFn[*missingTracks] {
	return func(in *missingTracks) (*missingTracks, error) {
		err := ds.WithTx(func(tx model.DataStore) error {
			for _, ms := range in.missing {
				for _, mt := range in.matched {
					if ms.Hash() == mt.Hash() {
						log.Debug(ctx, "Scanner: Found missing track", "missing", ms.Path, "matched", mt.Path)
						discardedID := ms.ID
						ms.ID = mt.ID
						ms.Missing = false
						ms.Path = mt.Path
						ms.FolderID = mt.FolderID
						err := tx.MediaFile(ctx).Put(&ms)
						if err != nil {
							return err
						}
						err = tx.MediaFile(ctx).Delete(discardedID)
						if err != nil {
							return err
						}
					}
				}
			}
			return nil
		})
		return nil, err
	}
}
