package lofty

import (
	"context"

	"github.com/navidrome/navidrome/core/metadataworker"
	"github.com/navidrome/navidrome/core/metadataworker/gen"
)

func extractViaGRPC(ctx context.Context, req request) (response, error) {
	files := make([]metadataworker.ExtractFile, len(req.Files))
	for i, file := range req.Files {
		files[i] = metadataworker.ExtractFile{Key: file.Key, Path: file.Path}
	}
	cfg := metadataworker.WorkerScanConfig{
		TagMappings:           req.TagMappings,
		ArtistSplitExceptions: req.ArtistSplitExceptions,
		ArtistsSplit:          req.ArtistsSplit,
		RolesSplit:            req.RolesSplit,
		ArtistJoiner:          req.ArtistJoiner,
		PIDConfig:             req.PIDConfig,
		LibraryID:             req.LibraryID,
	}
	resp, err := metadataworker.Extract(ctx, files, cfg)
	if err != nil {
		return response{}, err
	}
	return fromProtoExtract(resp), nil
}

func fromProtoExtract(resp *gen.ExtractResponse) response {
	out := response{
		Protocol: int(resp.GetProtocol()),
		Lofty:    resp.GetLofty(),
		Results:  make(map[string]rawResult, len(resp.GetResults())),
		Errors:   resp.GetErrors(),
	}
	if out.Protocol == 0 {
		out.Protocol = protocolVersion
	}
	if out.Lofty == "" {
		out.Lofty = loftyVersion
	}
	if out.Errors == nil {
		out.Errors = map[string]string{}
	}
	for key, meta := range resp.GetResults() {
		if meta == nil {
			continue
		}
		result := rawResult{
			Tags:          tagsFromProto(meta.GetTags()),
			DurationNS:    meta.GetDurationNs(),
			BitRate:       meta.GetBitRate(),
			BitDepth:      uint8(meta.GetBitDepth()),
			SampleRate:    meta.GetSampleRate(),
			Channels:      uint8(meta.GetChannels()),
			Codec:         meta.GetCodec(),
			HasPicture:    meta.GetHasPicture(),
			LyricsJSON:    meta.GetLyricsJson(),
			MediaFileJSON: meta.GetMediaFileJson(),
			CleanedTags:   tagsFromProto(meta.GetCleanedTags()),
		}
		if info := meta.GetFileInfo(); info != nil {
			fileInfo := &rawFileInfo{
				Name:       info.GetName(),
				Size:       info.GetSize(),
				ModifiedNS: info.GetModifiedNs(),
			}
			if info.GetHasCreatedNs() {
				created := info.GetCreatedNs()
				fileInfo.CreatedNS = &created
			}
			result.FileInfo = fileInfo
		}
		out.Results[key] = result
	}
	return out
}

func tagsFromProto(tags map[string]*gen.StringList) map[string][]string {
	if len(tags) == 0 {
		return nil
	}
	out := make(map[string][]string, len(tags))
	for key, list := range tags {
		if list == nil {
			continue
		}
		out[key] = append([]string(nil), list.Values...)
	}
	return out
}
