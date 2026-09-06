package metadataworker

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/navidrome/navidrome/core/metadataworker/gen"
	"github.com/navidrome/navidrome/core/rustworker"
	"google.golang.org/grpc"
)

var (
	metadataGRPC = rustworker.NewManagedGRPC(rustworker.ManagedGRPCConfig{
		Name:   "metadata",
		Listen: rustworker.DefaultListenAddr("navidrome-metadata"),
		Resolve: func() (string, error) {
			return Resolve()
		},
		Health: func(ctx context.Context, conn *grpc.ClientConn) error {
			_, err := gen.NewMetadataClient(conn).Health(ctx, &gen.HealthRequest{})
			return err
		},
	})
	errNoGRPC = errors.New("metadata gRPC worker unavailable")
)

// ErrNoGRPC is returned when the metadata gRPC worker is not running.
var ErrNoGRPC = errNoGRPC

// AdoptGRPC installs a preflight-started worker process into the managed host.
// Returns false when proc is nil or the host already has a live process.
func AdoptGRPC(proc *rustworker.GRPCProcess) bool {
	return metadataGRPC.Adopt(proc)
}

// WarmGRPC starts the metadata worker in the background for a warm first RPC.
func WarmGRPC() {
	metadataGRPC.Warm()
}

func callMetadata[T any](ctx context.Context, fn func(context.Context, *grpc.ClientConn) (T, error)) (T, error) {
	result, err := rustworker.CallGRPC(metadataGRPC, ctx, fn)
	if errors.Is(err, rustworker.ErrWorkerUnavailable) {
		var zero T
		return zero, errNoGRPC
	}
	return result, err
}

func tagsToProto(tags map[string][]string) map[string]*gen.StringList {
	if len(tags) == 0 {
		return nil
	}
	out := make(map[string]*gen.StringList, len(tags))
	for key, values := range tags {
		out[key] = &gen.StringList{Values: append([]string(nil), values...)}
	}
	return out
}

func tagsFromProto(tags map[string]*gen.StringList) map[string][]string {
	if len(tags) == 0 {
		return map[string][]string{}
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

func mappingsToProto(mappings map[string]TagMappingExport) map[string]*gen.TagMapping {
	if len(mappings) == 0 {
		return nil
	}
	out := make(map[string]*gen.TagMapping, len(mappings))
	for name, mapping := range mappings {
		out[name] = &gen.TagMapping{
			Aliases:   append([]string(nil), mapping.Aliases...),
			Type:      mapping.Type,
			MaxLength: int32(mapping.MaxLength),
			Split:     append([]string(nil), mapping.Split...),
			Album:     mapping.Album,
		}
	}
	return out
}

func Extract(ctx context.Context, files []ExtractFile, cfg WorkerScanConfig) (*gen.ExtractResponse, error) {
	return extractGRPC(ctx, files, cfg)
}

type ExtractFile struct {
	Key  string
	Path string
}

func extractGRPC(ctx context.Context, files []ExtractFile, cfg WorkerScanConfig) (*gen.ExtractResponse, error) {
	return callMetadata(ctx, func(ctx context.Context, conn *grpc.ClientConn) (*gen.ExtractResponse, error) {
		cli := gen.NewMetadataClient(conn)
		protoFiles := make([]*gen.ExtractFile, len(files))
		for i, file := range files {
			protoFiles[i] = &gen.ExtractFile{Key: file.Key, Path: file.Path}
		}
		var pidJSON string
		if len(cfg.PIDConfig) > 0 {
			raw, err := json.Marshal(cfg.PIDConfig)
			if err != nil {
				return nil, err
			}
			pidJSON = string(raw)
		}
		return cli.Extract(ctx, &gen.ExtractRequest{
			Files:                 protoFiles,
			TagMappings:           mappingsToProto(cfg.TagMappings),
			ArtistSplitExceptions: append([]string(nil), cfg.ArtistSplitExceptions...),
			ArtistsSplit:          append([]string(nil), cfg.ArtistsSplit...),
			RolesSplit:            append([]string(nil), cfg.RolesSplit...),
			ArtistJoiner:          cfg.ArtistJoiner,
			PidConfigJson:         pidJSON,
			LibraryId:             int32(cfg.LibraryID),
		})
	})
}

func cleanTagsGRPC(ctx context.Context, filePath string, tags map[string][]string, mappings map[string]TagMappingExport, exceptions []string) (map[string][]string, error) {
	return callMetadata(ctx, func(ctx context.Context, conn *grpc.ClientConn) (map[string][]string, error) {
		cli := gen.NewMetadataClient(conn)
		resp, err := cli.CleanTags(ctx, &gen.CleanTagsRequest{
			Path:                  filePath,
			Tags:                  tagsToProto(tags),
			Mappings:              mappingsToProto(mappings),
			ArtistSplitExceptions: exceptions,
		})
		if err != nil {
			return nil, err
		}
		if !resp.GetOk() {
			return nil, errors.New(nonEmpty(resp.GetError(), "Rust clean tags failed"))
		}
		return tagsFromProto(resp.GetTags()), nil
	})
}

func mapMediaGRPC(ctx context.Context, path string, tags map[string][]string, lyricsJSON string, cfg WorkerScanConfig) (string, error) {
	return callMetadata(ctx, func(ctx context.Context, conn *grpc.ClientConn) (string, error) {
		cli := gen.NewMetadataClient(conn)
		var pidJSON string
		if len(cfg.PIDConfig) > 0 {
			raw, err := json.Marshal(cfg.PIDConfig)
			if err != nil {
				return "", err
			}
			pidJSON = string(raw)
		}
		resp, err := cli.MapMedia(ctx, &gen.MapMediaRequest{
			Tags:                  tagsToProto(tags),
			Path:                  path,
			LyricsJson:            lyricsJSON,
			ArtistSplitExceptions: append([]string(nil), cfg.ArtistSplitExceptions...),
			ArtistsSplit:          append([]string(nil), cfg.ArtistsSplit...),
			RolesSplit:            append([]string(nil), cfg.RolesSplit...),
			ArtistJoiner:          cfg.ArtistJoiner,
			PidConfigJson:         pidJSON,
			LibraryId:             int32(cfg.LibraryID),
		})
		if err != nil {
			return "", err
		}
		if !resp.GetOk() {
			return "", errors.New(nonEmpty(resp.GetError(), "Rust map media failed"))
		}
		if resp.GetMediaFileJson() == "" {
			return "", errors.New("map media worker returned empty media_file_json")
		}
		return resp.GetMediaFileJson(), nil
	})
}

func parseLyricsGRPC(ctx context.Context, suffix, lang string, contents []byte) (string, error) {
	return callMetadata(ctx, func(ctx context.Context, conn *grpc.ClientConn) (string, error) {
		cli := gen.NewMetadataClient(conn)
		resp, err := cli.ParseLyrics(ctx, &gen.ParseLyricsRequest{Suffix: suffix, Lang: lang, Contents: contents})
		if err != nil {
			return "", err
		}
		if !resp.GetOk() {
			return "", errors.New(nonEmpty(resp.GetError(), "Rust lyrics worker could not parse lyrics"))
		}
		if resp.GetLyricsJson() == "" {
			return "[]", nil
		}
		return resp.GetLyricsJson(), nil
	})
}

func normalizeFTSGRPC(ctx context.Context, values []string) (string, error) {
	return callMetadata(ctx, func(ctx context.Context, conn *grpc.ClientConn) (string, error) {
		cli := gen.NewMetadataClient(conn)
		resp, err := cli.NormalizeFts(ctx, &gen.NormalizeFtsRequest{Values: values})
		if err != nil {
			return "", err
		}
		if !resp.GetOk() {
			return "", errors.New(nonEmpty(resp.GetError(), "Rust normalize worker failed"))
		}
		return resp.GetNormalized(), nil
	})
}

// NormalizeFtsBatch normalizes many value-groups in one local metadata RPC.
func NormalizeFtsBatch(ctx context.Context, groups [][]string) ([]string, error) {
	if len(groups) == 0 {
		return nil, nil
	}
	return callMetadata(ctx, func(ctx context.Context, conn *grpc.ClientConn) ([]string, error) {
		cli := gen.NewMetadataClient(conn)
		items := make([]*gen.StringList, len(groups))
		for i, values := range groups {
			items[i] = &gen.StringList{Values: append([]string(nil), values...)}
		}
		resp, err := cli.NormalizeFtsBatch(ctx, &gen.NormalizeFtsBatchRequest{Items: items})
		if err != nil {
			return nil, err
		}
		if !resp.GetOk() {
			return nil, errors.New(nonEmpty(resp.GetError(), "Rust normalize batch failed"))
		}
		out := resp.GetNormalized()
		if len(out) != len(groups) {
			return nil, errors.New("Rust normalize batch length mismatch")
		}
		return append([]string(nil), out...), nil
	})
}

func buildFTS5QueryGRPC(ctx context.Context, query string) (buildFTS5QueryResult, error) {
	return callMetadata(ctx, func(ctx context.Context, conn *grpc.ClientConn) (buildFTS5QueryResult, error) {
		cli := gen.NewMetadataClient(conn)
		resp, err := cli.BuildFts5Query(ctx, &gen.BuildFts5QueryRequest{Query: query})
		if err != nil {
			return buildFTS5QueryResult{}, err
		}
		if !resp.GetOk() {
			return buildFTS5QueryResult{}, errors.New(nonEmpty(resp.GetError(), "Rust build FTS5 query worker failed"))
		}
		return buildFTS5QueryResult{Query: resp.GetQuery(), Degraded: resp.GetDegraded()}, nil
	})
}

// ProcessImage is the gRPC image-resize/sniff entry used by the artwork package.
func ProcessImage(ctx context.Context, req *gen.ImageRequest) (*gen.ImageResponse, error) {
	return callMetadata(ctx, func(ctx context.Context, conn *grpc.ClientConn) (*gen.ImageResponse, error) {
		cli := gen.NewMetadataClient(conn)
		resp, err := cli.ProcessImage(ctx, req)
		if err != nil {
			return nil, err
		}
		if !resp.GetOk() {
			return nil, errors.New(nonEmpty(resp.GetError(), "Rust image worker request failed"))
		}
		return resp, nil
	})
}

func ExtractPicture(ctx context.Context, path string, maxBytes int64) ([]byte, error) {
	return callMetadata(ctx, func(ctx context.Context, conn *grpc.ClientConn) ([]byte, error) {
		cli := gen.NewMetadataClient(conn)
		resp, err := cli.ExtractPicture(ctx, &gen.ExtractPictureRequest{Path: path, MaxBytes: maxBytes})
		if err != nil {
			return nil, err
		}
		if !resp.GetOk() {
			return nil, errors.New(nonEmpty(resp.GetError(), "metadata worker could not extract artwork"))
		}
		return resp.GetBody(), nil
	})
}

func nonEmpty(v, fallback string) string {
	if v == "" {
		return fallback
	}
	return v
}
