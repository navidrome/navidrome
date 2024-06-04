package scanner2

import (
	"cmp"
	"context"
	"fmt"
	"slices"

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

type phaseMissingTracks struct {
	ctx context.Context
	ds  model.DataStore
}

func createPhaseMissingTracks(ctx context.Context, ds model.DataStore) *phaseMissingTracks {
	return &phaseMissingTracks{ctx: ctx, ds: ds}
}

func (p *phaseMissingTracks) producer() ppl.Producer[*missingTracks] {
	return ppl.NewProducer(p.produce, ppl.Name("load missing tracks from db"))
}

func (p *phaseMissingTracks) produce(put func(tracks *missingTracks)) error {
	var putIfMatched = func(mt missingTracks) {
		log.Trace(p.ctx, "Scanner: Found missing and matching tracks", "pid", mt.pid, "missing", len(mt.missing), "matched", len(mt.matched), "lib", mt.lib.Name)
		if mt.pid != "" && len(mt.matched) > 0 {
			put(&mt)
		}
	}
	libs, err := p.ds.Library(p.ctx).GetAll()
	if err != nil {
		return fmt.Errorf("error loading libraries: %w", err)
	}
	for _, lib := range libs {
		if lib.LastScanStartedAt.IsZero() {
			continue
		}
		mfs, err := p.ds.MediaFile(p.ctx).GetMissingAndMatching(lib.ID)
		if err != nil {
			return fmt.Errorf("error loading missing tracks for library %s: %w", lib.Name, err)
		}
		if len(mfs) == 0 {
			continue
		}
		slices.SortFunc(mfs, func(i, j model.MediaFile) int {
			return cmp.Compare(i.PID, j.PID)
		})
		mt := missingTracks{lib: lib}
		for _, mf := range mfs {
			if mt.pid != mf.PID {
				putIfMatched(mt)
				mt = missingTracks{lib: lib}
			}
			mt.pid = mf.PID
			if mf.Missing {
				mt.missing = append(mt.missing, mf)
			} else {
				mt.matched = append(mt.matched, mf)
			}
		}
		putIfMatched(mt)
	}

	return nil
}

func (p *phaseMissingTracks) stages() []ppl.Stage[*missingTracks] {
	return []ppl.Stage[*missingTracks]{
		ppl.NewStage(p.processMissingTracks, ppl.Name("process missing tracks")),
	}
}

func (p *phaseMissingTracks) processMissingTracks(in *missingTracks) (*missingTracks, error) {
	err := p.ds.WithTx(func(tx model.DataStore) error {
		for _, ms := range in.missing {
			var exactMatch model.MediaFile
			var equivalentMatch model.MediaFile

			// Identify exact and equivalent matches
			for _, mt := range in.matched {
				if ms.Equals(mt) {
					exactMatch = mt
					break // Prioritize exact match
				}
				if ms.IsEquivalent(mt) {
					equivalentMatch = mt
				}
			}

			// Process the exact match if found
			if exactMatch.ID != "" {
				log.Debug(p.ctx, "Scanner: Found missing track", "missing", ms.Path, "matched", exactMatch.Path, "lib", in.lib.Name)
				err := p.moveMatched(tx, exactMatch, ms)
				if err != nil {
					log.Error(p.ctx, "Scanner: Error moving matched track", "missing", ms.Path, "matched", exactMatch.Path, "lib", in.lib.Name, err)
					return err
				}
				continue
			}

			// Process the equivalent match if no exact match was found
			if equivalentMatch.ID != "" {
				log.Debug(p.ctx, "Scanner: Found upgraded track with same tags", "missing", ms.Path, "matched", equivalentMatch.Path, "lib", in.lib.Name)
				err := p.moveMatched(tx, equivalentMatch, ms)
				if err != nil {
					log.Error(p.ctx, "Scanner: Error moving upgraded track", "missing", ms.Path, "matched", equivalentMatch.Path, "lib", in.lib.Name, err)
					return err
				}
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return in, nil
}

func (p *phaseMissingTracks) moveMatched(tx model.DataStore, mt, ms model.MediaFile) error {
	discardedID := mt.ID
	mt.ID = ms.ID
	err := tx.MediaFile(p.ctx).Put(&mt)
	if err != nil {
		return fmt.Errorf("update matched track: %w", err)
	}
	return tx.MediaFile(p.ctx).Delete(discardedID)
}

func (p *phaseMissingTracks) finalize(error) error {
	return nil
}

var _ phase[*missingTracks] = (*phaseMissingTracks)(nil)
