package playlists

import (
	"cmp"
	"context"
	"fmt"
	"io"
	"net/url"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/utils/slice"
	"golang.org/x/text/unicode/norm"
)

func (s *playlists) parseM3U(ctx context.Context, pls *model.Playlist, folder *model.Folder, reader io.Reader) error {
	mediaFileRepository := s.ds.MediaFile(ctx)
	resolver, err := newPathResolver(ctx, s.ds)
	if err != nil {
		return err
	}
	var mfs model.MediaFiles
	// Chunk size of 100 lines, as each line can generate up to 4 lookup candidates
	// (NFC/NFD × raw/lowercase), and SQLite has a max expression tree depth of 1000.
	for lines := range slice.CollectChunks(slice.LinesFrom(reader), 100) {
		filteredLines := make([]string, 0, len(lines))
		for _, line := range lines {
			line := strings.TrimSpace(line)
			if strings.HasPrefix(line, "#PLAYLIST:") {
				pls.Name = line[len("#PLAYLIST:"):]
				continue
			}
			// Skip empty lines and extended info
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			if after, ok := strings.CutPrefix(line, "file://"); ok {
				line = after
				line, _ = url.QueryUnescape(line)
			}
			if !model.IsAudioFile(line) {
				continue
			}
			filteredLines = append(filteredLines, line)
		}
		resolvedPaths, err := resolver.resolvePaths(ctx, folder, filteredLines)
		if err != nil {
			log.Warn(ctx, "Error resolving paths in playlist", "playlist", pls.Name, err)
			continue
		}

		// SQLite comparisons do not perform Unicode normalization, and filesystem normalization
		// differs across platforms (macOS often yields NFD, while Linux/Windows typically use NFC).
		// Generate lookup candidates for both forms so playlist entries match DB paths regardless
		// of the original normalization. See https://github.com/navidrome/navidrome/issues/4884
		//
		// We also include the original (non-lowercased) paths because SQLite's COLLATE NOCASE
		// only handles ASCII case-insensitivity. Non-ASCII characters like fullwidth letters
		// (e.g., ＡＢＣＤ vs ａｂｃｄ) are not matched case-insensitively by NOCASE.
		lookupCandidates := make([]string, 0, len(resolvedPaths)*4)
		seen := make(map[string]struct{}, len(resolvedPaths)*4)
		for _, path := range resolvedPaths {
			// Add original paths first (for exact matching of non-ASCII characters)
			nfcRaw := norm.NFC.String(path)
			if _, ok := seen[nfcRaw]; !ok {
				seen[nfcRaw] = struct{}{}
				lookupCandidates = append(lookupCandidates, nfcRaw)
			}
			nfdRaw := norm.NFD.String(path)
			if _, ok := seen[nfdRaw]; !ok {
				seen[nfdRaw] = struct{}{}
				lookupCandidates = append(lookupCandidates, nfdRaw)
			}

			// Add lowercased paths (for ASCII case-insensitive matching via NOCASE)
			nfc := strings.ToLower(nfcRaw)
			if _, ok := seen[nfc]; !ok {
				seen[nfc] = struct{}{}
				lookupCandidates = append(lookupCandidates, nfc)
			}
			nfd := strings.ToLower(nfdRaw)
			if _, ok := seen[nfd]; !ok {
				seen[nfd] = struct{}{}
				lookupCandidates = append(lookupCandidates, nfd)
			}
		}

		found, err := mediaFileRepository.FindByPaths(lookupCandidates)
		if err != nil {
			log.Warn(ctx, "Error reading files from DB", "playlist", pls.Name, err)
			continue
		}

		// Build lookup map with library-qualified keys, normalized for comparison.
		// Canonicalize to NFC so NFD/NFC become comparable.
		existing := make(map[string]int, len(found))
		for idx := range found {
			key := fmt.Sprintf("%d:%s", found[idx].LibraryID, strings.ToLower(norm.NFC.String(found[idx].Path)))
			existing[key] = idx
		}

		// Find media files in the order of the resolved paths, to keep playlist order.
		// Both `existing` keys and `resolvedPaths` use the library-qualified format "libraryID:relativePath",
		// so normalizing the full string produces matching keys (digits and ':' are ASCII-invariant).
		for _, path := range resolvedPaths {
			key := strings.ToLower(norm.NFC.String(path))
			idx, ok := existing[key]
			if ok {
				mfs = append(mfs, found[idx])
			} else {
				// Prefer logging a composed representation when possible to avoid confusing output
				// with decomposed combining marks.
				log.Warn(ctx, "Path in playlist not found", "playlist", pls.Name, "path", norm.NFC.String(path))
			}
		}
	}
	if pls.Name == "" {
		pls.Name = time.Now().Format(time.RFC3339)
	}
	pls.Tracks = nil
	pls.AddMediaFiles(mfs)

	return nil
}

// pathResolution holds the result of resolving a playlist path to a library-relative path.
type pathResolution struct {
	absolutePath string
	libraryPath  string
	libraryID    int
	valid        bool
}

// ToQualifiedString converts the path resolution to a library-qualified string with forward slashes.
// Format: "libraryID:relativePath" with forward slashes for path separators.
func (r pathResolution) ToQualifiedString() (string, error) {
	if !r.valid {
		return "", fmt.Errorf("invalid path resolution")
	}
	relativePath, err := filepath.Rel(r.libraryPath, r.absolutePath)
	if err != nil {
		return "", err
	}
	// Convert path separators to forward slashes
	return fmt.Sprintf("%d:%s", r.libraryID, filepath.ToSlash(relativePath)), nil
}

// libraryMatcher holds sorted libraries with cleaned paths for efficient path matching.
type libraryMatcher struct {
	libraries    model.Libraries
	cleanedPaths []string
}

// findLibraryForPath finds which library contains the given absolute path.
// Returns library ID and path, or 0 and empty string if not found.
func (lm *libraryMatcher) findLibraryForPath(absolutePath string) (int, string) {
	// Check sorted libraries (longest path first) to find the best match
	for i, cleanLibPath := range lm.cleanedPaths {
		// Check if absolutePath is under this library path
		if strings.HasPrefix(absolutePath, cleanLibPath) {
			// Ensure it's a proper path boundary (not just a prefix)
			if len(absolutePath) == len(cleanLibPath) || absolutePath[len(cleanLibPath)] == filepath.Separator {
				return lm.libraries[i].ID, cleanLibPath
			}
		}
	}
	return 0, ""
}

// newLibraryMatcher creates a libraryMatcher with libraries sorted by path length (longest first).
// This ensures correct matching when library paths are prefixes of each other.
// Example: /music-classical must be checked before /music
// Otherwise, /music-classical/track.mp3 would match /music instead of /music-classical
func newLibraryMatcher(libs model.Libraries) *libraryMatcher {
	// Sort libraries by path length (descending) to ensure longest paths match first.
	slices.SortFunc(libs, func(i, j model.Library) int {
		return cmp.Compare(len(j.Path), len(i.Path)) // Reverse order for descending
	})

	// Pre-clean all library paths once for efficient matching
	cleanedPaths := make([]string, len(libs))
	for i, lib := range libs {
		cleanedPaths[i] = filepath.Clean(lib.Path)
	}
	return &libraryMatcher{
		libraries:    libs,
		cleanedPaths: cleanedPaths,
	}
}

// pathResolver handles path resolution logic for playlist imports.
type pathResolver struct {
	matcher *libraryMatcher
}

// newPathResolver creates a pathResolver with libraries loaded from the datastore.
func newPathResolver(ctx context.Context, ds model.DataStore) (*pathResolver, error) {
	libs, err := ds.Library(ctx).GetAll()
	if err != nil {
		return nil, err
	}
	matcher := newLibraryMatcher(libs)
	return &pathResolver{matcher: matcher}, nil
}

// resolvePath determines the absolute path and library path for a playlist entry.
// For absolute paths, it uses them directly.
// For relative paths, it resolves them relative to the playlist's folder location.
// Example: playlist at /music/playlists/test.m3u with line "../songs/abc.mp3"
//
//	resolves to /music/songs/abc.mp3
func (r *pathResolver) resolvePath(line string, folder *model.Folder) pathResolution {
	var absolutePath string
	if folder != nil && !filepath.IsAbs(line) {
		// Resolve relative path to absolute path based on playlist location
		absolutePath = filepath.Clean(filepath.Join(folder.AbsolutePath(), line))
	} else {
		// Use absolute path directly after cleaning
		absolutePath = filepath.Clean(line)
	}

	return r.findInLibraries(absolutePath)
}

// findInLibraries matches an absolute path against all known libraries and returns
// a pathResolution with the library information. Returns an invalid resolution if
// the path is not found in any library.
func (r *pathResolver) findInLibraries(absolutePath string) pathResolution {
	libID, libPath := r.matcher.findLibraryForPath(absolutePath)
	if libID == 0 {
		return pathResolution{valid: false}
	}
	return pathResolution{
		absolutePath: absolutePath,
		libraryPath:  libPath,
		libraryID:    libID,
		valid:        true,
	}
}

// resolvePaths converts playlist file paths to library-qualified paths (format: "libraryID:relativePath").
// For relative paths, it resolves them to absolute paths first, then determines which
// library they belong to. This allows playlists to reference files across library boundaries.
func (r *pathResolver) resolvePaths(ctx context.Context, folder *model.Folder, lines []string) ([]string, error) {
	results := make([]string, 0, len(lines))
	for idx, line := range lines {
		resolution := r.resolvePath(line, folder)

		if !resolution.valid {
			log.Warn(ctx, "Path in playlist not found in any library", "path", line, "line", idx)
			continue
		}

		qualifiedPath, err := resolution.ToQualifiedString()
		if err != nil {
			log.Debug(ctx, "Error getting library-qualified path", "path", line,
				"libPath", resolution.libraryPath, "filePath", resolution.absolutePath, err)
			continue
		}

		results = append(results, qualifiedPath)
	}

	return results, nil
}
