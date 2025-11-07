package scanner

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"

	ppl "github.com/google/go-pipeline/pkg/pipeline"
	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/consts"
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
	ctx                       context.Context
	ds                        model.DataStore
	totalMatched              atomic.Uint32
	state                     *scanState
	processedAlbumAnnotations map[string]bool // Track processed album annotation reassignments
	annotationMutex           sync.RWMutex    // Protects processedAlbumAnnotations
}

func createPhaseMissingTracks(ctx context.Context, state *scanState, ds model.DataStore) *phaseMissingTracks {
	return &phaseMissingTracks{
		ctx:                       ctx,
		ds:                        ds,
		state:                     state,
		processedAlbumAnnotations: make(map[string]bool),
	}
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
		if mt.pid != "" && len(mt.missing) > 0 {
			log.Trace(p.ctx, "Scanner: Found missing tracks", "pid", mt.pid, "missing", "title", mt.missing[0].Title,
				len(mt.missing), "matched", len(mt.matched), "lib", mt.lib.Name,
			)
			count++
			put(&mt)
		}
	}
	for _, lib := range p.state.libraries {
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
		ppl.NewStage(p.processCrossLibraryMoves, ppl.Name("process cross-library moves")),
	}
}

func (p *phaseMissingTracks) processMissingTracks(in *missingTracks) (*missingTracks, error) {
	hasMatches := false

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
			hasMatches = true
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
			hasMatches = true
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
			hasMatches = true
		}
	}

	// If any matches were found in this missingTracks group, return nil
	// This signals the next stage to skip processing this group
	if hasMatches {
		return nil, nil
	}

	// If no matches found, pass through to next stage
	return in, nil
}

// processCrossLibraryMoves processes files that weren't matched within their library
// and attempts to find matches in other libraries
func (p *phaseMissingTracks) processCrossLibraryMoves(in *missingTracks) (*missingTracks, error) {
	// Skip if input is nil (meaning previous stage found matches)
	if in == nil {
		return nil, nil
	}

	log.Debug(p.ctx, "Scanner: Processing cross-library moves", "pid", in.pid, "missing", len(in.missing), "lib", in.lib.Name)

	for _, missing := range in.missing {
		found, err := p.findCrossLibraryMatch(missing)
		if err != nil {
			log.Error(p.ctx, "Scanner: Error searching for cross-library matches", "missing", missing.Path, "lib", in.lib.Name, err)
			continue
		}

		if found.ID != "" {
			log.Debug(p.ctx, "Scanner: Found cross-library moved track", "missing", missing.Path, "movedTo", found.Path, "fromLib", in.lib.Name, "toLib", found.LibraryName)
			err := p.moveMatched(found, missing)
			if err != nil {
				log.Error(p.ctx, "Scanner: Error moving cross-library track", "missing", missing.Path, "movedTo", found.Path, err)
				continue
			}
			p.totalMatched.Add(1)
		}
	}

	return in, nil
}

// findCrossLibraryMatch searches for a missing file in other libraries using two-tier matching
func (p *phaseMissingTracks) findCrossLibraryMatch(missing model.MediaFile) (model.MediaFile, error) {
	// First tier: Search by MusicBrainz Track ID if available
	if missing.MbzReleaseTrackID != "" {
		matches, err := p.ds.MediaFile(p.ctx).FindRecentFilesByMBZTrackID(missing, missing.CreatedAt)
		if err != nil {
			log.Error(p.ctx, "Scanner: Error searching for recent files by MBZ Track ID", "mbzTrackID", missing.MbzReleaseTrackID, err)
		} else {
			// Apply the same matching logic as within-library matching
			for _, match := range matches {
				if missing.Equals(match) {
					return match, nil // Exact match found
				}
			}

			// If only one match and it's equivalent, use it
			if len(matches) == 1 && missing.IsEquivalent(matches[0]) {
				return matches[0], nil
			}
		}
	}

	// Second tier: Search by intrinsic properties (title, size, suffix, etc.)
	matches, err := p.ds.MediaFile(p.ctx).FindRecentFilesByProperties(missing, missing.CreatedAt)
	if err != nil {
		log.Error(p.ctx, "Scanner: Error searching for recent files by properties", "missing", missing.Path, err)
		return model.MediaFile{}, err
	}

	// Apply the same matching logic as within-library matching
	for _, match := range matches {
		if missing.Equals(match) {
			return match, nil // Exact match found
		}
	}

	// If only one match and it's equivalent, use it
	if len(matches) == 1 && missing.IsEquivalent(matches[0]) {
		return matches[0], nil
	}

	return model.MediaFile{}, nil
}

func (p *phaseMissingTracks) moveMatched(target, missing model.MediaFile) error {
	return p.ds.WithTx(func(tx model.DataStore) error {
		discardedID := target.ID
		oldAlbumID := missing.AlbumID
		newAlbumID := target.AlbumID

		// Update the target media file with the missing file's ID. This effectively "moves" the track
		// to the new location while keeping its annotations and references intact.
		target.ID = missing.ID
		err := tx.MediaFile(p.ctx).Put(&target)
		if err != nil {
			return fmt.Errorf("update matched track: %w", err)
		}

		// Discard the new mediafile row (the one that was moved to)
		err = tx.MediaFile(p.ctx).Delete(discardedID)
		if err != nil {
			return fmt.Errorf("delete discarded track: %w", err)
		}

		// Handle album annotation reassignment if AlbumID changed
		if oldAlbumID != newAlbumID {
			// Use newAlbumID as key since we only care about avoiding duplicate reassignments to the same target
			p.annotationMutex.RLock()
			alreadyProcessed := p.processedAlbumAnnotations[newAlbumID]
			p.annotationMutex.RUnlock()

			if !alreadyProcessed {
				p.annotationMutex.Lock()
				// Double-check pattern to avoid race conditions
				if !p.processedAlbumAnnotations[newAlbumID] {
					// Reassign direct album annotations (starred, rating)
					log.Debug(p.ctx, "Scanner: Reassigning album annotations", "from", oldAlbumID, "to", newAlbumID)
					if err := tx.Album(p.ctx).ReassignAnnotation(oldAlbumID, newAlbumID); err != nil {
						log.Warn(p.ctx, "Scanner: Could not reassign album annotations", "from", oldAlbumID, "to", newAlbumID, err)
					}

					// Note: RefreshPlayCounts will be called in later phases, so we don't need to call it here
					p.processedAlbumAnnotations[newAlbumID] = true
				}
				p.annotationMutex.Unlock()
			} else {
				log.Trace(p.ctx, "Scanner: Skipping album annotation reassignment", "from", oldAlbumID, "to", newAlbumID)
			}
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
	if err != nil {
		return err
	}

	// Check if we should purge missing items
	if conf.Server.Scanner.PurgeMissing == consts.PurgeMissingAlways || (conf.Server.Scanner.PurgeMissing == consts.PurgeMissingFull && p.state.fullScan) {
		if err = p.purgeMissing(); err != nil {
			log.Error(p.ctx, "Scanner: Error purging missing items", err)
		}
	}

	return err
}

func (p *phaseMissingTracks) purgeMissing() error {
	deletedCount, err := p.ds.MediaFile(p.ctx).DeleteAllMissing()
	if err != nil {
		return fmt.Errorf("error deleting missing files: %w", err)
	}

	if deletedCount > 0 {
		log.Info(p.ctx, "Scanner: Purged missing items from the database", "mediaFiles", deletedCount)
		// Set changesDetected to true so that garbage collection will run at the end of the scan process
		p.state.changesDetected.Store(true)
	} else {
		log.Debug(p.ctx, "Scanner: No missing items to purge")
	}

	return nil
}

var _ phase[*missingTracks] = (*phaseMissingTracks)(nil)
