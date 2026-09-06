// Package lofty provides the local metadata extractor backed by the Rust Lofty worker.
//
// Extract uses the metadata gRPC worker only.
package lofty

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"
	"time"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/core/metadataworker"
	"github.com/navidrome/navidrome/core/storage/local"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/metadata"
)

const (
	protocolVersion = 1
	loftyVersion    = "0.25.1"
)

type request struct {
	Files                 []inputFile                                `json:"files"`
	TagMappings           map[string]metadataworker.TagMappingExport `json:"tag_mappings,omitempty"`
	ArtistSplitExceptions []string                                   `json:"artist_split_exceptions,omitempty"`
	ArtistsSplit          []string                                   `json:"artists_split,omitempty"`
	RolesSplit            []string                                   `json:"roles_split,omitempty"`
	ArtistJoiner          string                                     `json:"artist_joiner,omitempty"`
	PIDConfig             map[string]any                             `json:"pid_config,omitempty"`
	LibraryID             int                                        `json:"library_id,omitempty"`
}

type inputFile struct {
	Key  string `json:"key"`
	Path string `json:"path"`
}

type response struct {
	Protocol int                  `json:"protocol"`
	Lofty    string               `json:"lofty"`
	Results  map[string]rawResult `json:"results"`
	Errors   map[string]string    `json:"errors"`
}

type rawResult struct {
	Tags          map[string][]string `json:"tags"`
	FileInfo      *rawFileInfo        `json:"file_info,omitempty"`
	DurationNS    uint64              `json:"duration_ns"`
	BitRate       uint32              `json:"bit_rate"`
	BitDepth      uint8               `json:"bit_depth"`
	SampleRate    uint32              `json:"sample_rate"`
	Channels      uint8               `json:"channels"`
	Codec         string              `json:"codec"`
	HasPicture    bool                `json:"has_picture"`
	LyricsJSON    string              `json:"lyrics_json,omitempty"`
	MediaFileJSON string              `json:"media_file_json,omitempty"`
	CleanedTags   map[string][]string `json:"cleaned_tags,omitempty"`
}

type rawFileInfo struct {
	Name       string `json:"name"`
	Size       uint64 `json:"size"`
	ModifiedNS int64  `json:"modified_ns"`
	CreatedNS  *int64 `json:"created_ns,omitempty"`
}

type workerFileInfo struct {
	name      string
	size      int64
	modified  time.Time
	birthTime time.Time
}

func (f workerFileInfo) Name() string         { return f.name }
func (f workerFileInfo) Size() int64          { return f.size }
func (f workerFileInfo) Mode() fs.FileMode    { return 0 }
func (f workerFileInfo) ModTime() time.Time   { return f.modified }
func (f workerFileInfo) IsDir() bool          { return false }
func (f workerFileInfo) Sys() any             { return nil }
func (f workerFileInfo) BirthTime() time.Time { return f.birthTime }

type extractor struct {
	baseDir string
}

func (e *extractor) Parse(files ...string) (map[string]metadata.Info, error) {
	return e.ParseContext(context.Background(), files...)
}

func (e *extractor) ParseContext(ctx context.Context, files ...string) (map[string]metadata.Info, error) {
	if len(files) == 0 {
		return map[string]metadata.Info{}, nil
	}

	req, err := e.buildRequest(ctx, files)
	if err != nil {
		return nil, err
	}

	resp, err := extractViaGRPC(ctx, req)
	if err != nil {
		return nil, err
	}
	return convertResponse(resp)
}

func (e *extractor) Version() string {
	return loftyVersion
}

func (e *extractor) buildRequest(ctx context.Context, files []string) (request, error) {
	base, err := filepath.Abs(e.baseDir)
	if err != nil {
		return request{}, fmt.Errorf("resolving music root: %w", err)
	}

	inputs := make([]inputFile, 0, len(files))
	for _, key := range files {
		if key == "" {
			continue
		}
		candidate := filepath.Clean(filepath.Join(base, filepath.FromSlash(key)))
		rel, err := filepath.Rel(base, candidate)
		if err != nil {
			return request{}, fmt.Errorf("resolving metadata path %q: %w", key, err)
		}
		if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || filepath.IsAbs(rel) {
			return request{}, fmt.Errorf("metadata path escapes music root: %q", key)
		}
		inputs = append(inputs, inputFile{Key: key, Path: candidate})
	}
	scanConfig := workerScanConfig(metadataworker.LibraryIDFromContext(ctx))
	return request{
		Files:                 inputs,
		TagMappings:           scanConfig.TagMappings,
		ArtistSplitExceptions: scanConfig.ArtistSplitExceptions,
		ArtistsSplit:          scanConfig.ArtistsSplit,
		RolesSplit:            scanConfig.RolesSplit,
		ArtistJoiner:          scanConfig.ArtistJoiner,
		PIDConfig:             scanConfig.PIDConfig,
		LibraryID:             scanConfig.LibraryID,
	}, nil
}

func convertResponse(resp response) (map[string]metadata.Info, error) {
	if requestErr := resp.Errors["$request"]; requestErr != "" {
		return nil, errors.New(requestErr)
	}
	results := make(map[string]metadata.Info, len(resp.Results))
	for key, value := range resp.Results {
		fileInfo, err := convertFileInfo(value.FileInfo)
		if err != nil {
			return nil, fmt.Errorf("invalid file information for %q: %w", key, err)
		}
		cleaned := model.Tags{}
		for tag, values := range value.CleanedTags {
			cleaned[model.TagName(tag)] = append([]string(nil), values...)
		}
		results[key] = metadata.Info{
			FileInfo: fileInfo,
			Tags:     value.Tags,
			AudioProperties: metadata.AudioProperties{
				Duration:   time.Duration(value.DurationNS),
				BitRate:    int(value.BitRate),
				BitDepth:   int(value.BitDepth),
				SampleRate: int(value.SampleRate),
				Channels:   int(value.Channels),
				Codec:      value.Codec,
			},
			HasPicture:    value.HasPicture,
			LyricsJSON:    value.LyricsJSON,
			MediaFileJSON: value.MediaFileJSON,
			CleanedTags:   cleaned,
		}
	}
	for key, workerErr := range resp.Errors {
		if key == "$request" {
			continue
		}
		log.Warn("Lofty could not read metadata; skipping file", "filePath", key, "error", workerErr)
	}
	return results, nil
}

func convertFileInfo(raw *rawFileInfo) (metadata.FileInfo, error) {
	// Older administrator-provided workers do not return file_info. Keep the
	// existing Go stat fallback in localFS for protocol compatibility.
	if raw == nil {
		return nil, nil
	}
	if raw.Size > uint64(1<<63-1) {
		return nil, fmt.Errorf("file size %d exceeds int64", raw.Size)
	}
	modified := time.Unix(0, raw.ModifiedNS).UTC()
	birthTime := modified
	if raw.CreatedNS != nil {
		birthTime = time.Unix(0, *raw.CreatedNS).UTC()
	}
	return workerFileInfo{
		name:      raw.Name,
		size:      int64(raw.Size),
		modified:  modified,
		birthTime: birthTime,
	}, nil
}

func resolveWorkerPath() string {
	if workerPath, err := metadataworker.Resolve(); err == nil {
		return workerPath
	}
	return metadataworker.BinaryName()
}

var _ local.Extractor = (*extractor)(nil)

func init() {
	local.RegisterExtractor("lofty", func(_ fs.FS, baseDir string) local.Extractor {
		return &extractor{baseDir: baseDir}
	})
	conf.AddHook(func() {
		log.Debug("Lofty metadata extractor", "version", loftyVersion, "worker", resolveWorkerPath())
	})
}
