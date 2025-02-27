package scanner

import (
	"context"
	"fmt"
	"sync/atomic"

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

// phaseMissingTracks is responsible for processing missing media files during the scan process.
// It identifies media files that are marked as missing and attempts to find matching files that
// may have been moved or renamed. This phase helps in maintaining the integrity of the media
// library by ensuring that moved or renamed files are correctly updated in the database.
//
// The phaseMissingTracks phase performs the following steps:
// 1. Loads all libraries and their missing media files from the database.
// 2. For each library, it sorts the missing files by their PID (persistent identifier).
// 3. Groups missing and matched files by their PID and processes them to find exact or equivalent matches.
// 4. Updates the database with the new locations of the matched files and removes the old entries.
// 5. Logs the results and finalizes the phase by reporting the total number of matched files.
type phaseMissingTracks struct {
	ctx          context.Context
	ds           model.DataStore
	totalMatched atomic.Uint32
	state        *scanState
}

func createPhaseMissingTracks(ctx context.Context, state *scanState, ds model.DataStore) *phaseMissingTracks {
	return &phaseMissingTracks{ctx: ctx, ds: ds, state: state}
}

func (p *phaseMissingTracks) description() string {
	return "Process missing files, checking for moves"
}

func (p *phaseMissingTracks) producer() ppl.Producer[*missingTracks] {
	return ppl.NewProducer(p.produce, ppl.Name("load missing tracks from db"))
}

func (p *phaseMissingTracks) produce(put func(tracks *missingTracks)) error {
	count := 0
	var putIfMatched = func(mt missingTracks) {
		if mt.pid != "" && len(mt.matched) > 0 {
			log.Trace(p.ctx, "Scanner: Found missing and matching tracks", "pid", mt.pid, "missing", len(mt.missing), "matched", len(mt.matched), "lib", mt.lib.Name)
			count++
			put(&mt)
		}
	}
	libs, err := p.ds.Library(p.ctx).GetAll()
	if err != nil {
		return fmt.Errorf("loading libraries: %w", err)
	}
	for _, lib := range libs {
		if lib.LastScanStartedAt.IsZero() {
			continue
		}
		log.Debug(p.ctx, "Scanner: Checking missing tracks", "libraryId", lib.ID, "libraryName", lib.Name)
		cursor, err := p.ds.MediaFile(p.ctx).GetMissingAndMatching(lib.ID)
		if err != nil {
			return fmt.Errorf("loading missing tracks for library %s: %w", lib.Name, err)
		}

		// Group missing and matched tracks by PID
		mt := missingTracks{lib: lib}
		for mf, err := range cursor {
			if err != nil {
				return fmt.Errorf("loading missing tracks for library %s: %w", lib.Name, err)
			}
			if mt.pid != mf.PID {
				putIfMatched(mt)
				mt.pid = mf.PID
				mt.missing = nil
				mt.matched = nil
			}
			if mf.Missing {
				mt.missing = append(mt.missing, mf)
			} else {
				mt.matched = append(mt.matched, mf)
			}
		}
		putIfMatched(mt)
		if count == 0 {
			log.Debug(p.ctx, "Scanner: No potential moves found", "libraryId", lib.ID, "libraryName", lib.Name)
		} else {
			log.Debug(p.ctx, "Scanner: Found potential moves", "libraryId", lib.ID, "count", count)
		}
	}

	return nil
}

func (p *phaseMissingTracks) stages() []ppl.Stage[*missingTracks] {
	return []ppl.Stage[*missingTracks]{
		ppl.NewStage(p.processMissingTracks, ppl.Name("process missing tracks")),
	}
}

func (p *phaseMissingTracks) processMissingTracks(in *missingTracks) (*missingTracks, error) {
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

		// Use the exact match if found
		if exactMatch.ID != "" {
			log.Debug(p.ctx, "Scanner: Found missing track in a new place", "missing", ms.Path, "movedTo", exactMatch.Path, "lib", in.lib.Name)
			err := p.moveMatched(exactMatch, ms)
			if err != nil {
				log.Error(p.ctx, "Scanner: Error moving matched track", "missing", ms.Path, "movedTo", exactMatch.Path, "lib", in.lib.Name, err)
				return nil, err
			}
			p.totalMatched.Add(1)
			continue
		}

		// If there is only one missing and one matched track, consider them equivalent (same PID)
		if len(in.missing) == 1 && len(in.matched) == 1 {
			singleMatch := in.matched[0]
			log.Debug(p.ctx, "Scanner: Found track with same persistent ID in a new place", "missing", ms.Path, "movedTo", singleMatch.Path, "lib", in.lib.Name)
			err := p.moveMatched(singleMatch, ms)
			if err != nil {
				log.Error(p.ctx, "Scanner: Error updating matched track", "missing", ms.Path, "movedTo", singleMatch.Path, "lib", in.lib.Name, err)
				return nil, err
			}
			p.totalMatched.Add(1)
			continue
		}

		// Use the equivalent match if no other better match was found
		if equivalentMatch.ID != "" {
			log.Debug(p.ctx, "Scanner: Found missing track with same base path", "missing", ms.Path, "movedTo", equivalentMatch.Path, "lib", in.lib.Name)
			err := p.moveMatched(equivalentMatch, ms)
			if err != nil {
				log.Error(p.ctx, "Scanner: Error updating matched track", "missing", ms.Path, "movedTo", equivalentMatch.Path, "lib", in.lib.Name, err)
				return nil, err
			}
			p.totalMatched.Add(1)
		}
	}
	return in, nil
}

func (p *phaseMissingTracks) moveMatched(mt, ms model.MediaFile) error {
	return p.ds.WithTx(func(tx model.DataStore) error {
		discardedID := mt.ID
		mt.ID = ms.ID
		err := tx.MediaFile(p.ctx).Put(&mt)
		if err != nil {
			return fmt.Errorf("update matched track: %w", err)
		}
		err = tx.MediaFile(p.ctx).Delete(discardedID)
		if err != nil {
			return fmt.Errorf("delete discarded track: %w", err)
		}
		p.state.changesDetected.Store(true)
		return nil
	})
}

func (p *phaseMissingTracks) finalize(err error) error {
	matched := p.totalMatched.Load()
	if matched > 0 {
		log.Info(p.ctx, "Scanner: Found moved files", "total", matched, err)
	}
	return err
}

var _ phase[*missingTracks] = (*phaseMissingTracks)(nil)
