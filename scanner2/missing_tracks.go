package scanner2

import (
	"context"
	"path"

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
					// Check if the missing track is the exact same as one of the matched tracks
					if ms.Hash() == mt.Hash() {
						log.Debug(ctx, "Scanner: Found missing track", "missing", ms.Path, "matched", mt.Path, "lib", in.lib.Name)
						err := moveMatched(ctx, tx, mt, ms)
						if err != nil {
							log.Error(ctx, "Scanner: Error moving matched track", "missing", ms.Path, "matched", mt.Path, "lib", in.lib.Name, err)
							return err
						}
						continue
					}
					// Check if the missing track has the same tags and filename as one of the matched tracks
					if ms.Tags.Hash() == mt.Tags.Hash() && baseName(ms.Path) == baseName(mt.Path) {
						log.Debug(ctx, "Scanner: Found upgraded track with same tags", "missing", ms.Path, "matched", mt.Path, "lib", in.lib.Name)
						err := moveMatched(ctx, tx, mt, ms)
						if err != nil {
							log.Error(ctx, "Scanner: Error moving upgraded track", "missing", ms.Path, "matched", mt.Path, "lib", in.lib.Name, err)
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

func moveMatched(ctx context.Context, tx model.DataStore, mt model.MediaFile, ms model.MediaFile) error {
	discardedID := mt.ID
	mt.ID = ms.ID
	err := tx.MediaFile(ctx).Put(&mt)
	if err != nil {
		return err
	}
	return tx.MediaFile(ctx).Delete(discardedID)
}

func baseName(filePath string) string {
	p := path.Base(filePath)
	ext := path.Ext(p)
	return p[:len(p)-len(ext)]
}
