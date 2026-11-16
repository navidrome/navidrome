package plugins

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/plugins/capabilities"
)

const CapabilityLyrics Capability = "Lyrics"

const (
	FuncLyricsGetLyrics   = "nd_lyrics_get_lyrics"
	FuncLyricsGetLyricsV2 = "nd_lyrics_get_lyrics_v2"
	FuncLyricsGetVariants = "nd_lyrics_get_variants"
	FuncLyricsGetOriginal = "nd_lyrics_get_original"
)

// maxConcurrentLyricsCalls caps in-flight lyrics calls per plugin: clients prefetch
// lyrics for whole queues, and the resulting burst can rate-limit upstream providers.
const maxConcurrentLyricsCalls = 2

func init() {
	registerCapability(
		CapabilityLyrics,
		FuncLyricsGetLyrics,
	)
}

func newLyricsPlugin(p *plugin) *LyricsPlugin {
	displayName := p.name
	if p.manifest != nil && strings.TrimSpace(p.manifest.Name) != "" {
		displayName = strings.TrimSpace(p.manifest.Name)
	}
	return &LyricsPlugin{name: p.name, displayName: displayName, plugin: p}
}

// LyricsPlugin adapts a WASM plugin with the Lyrics capability.
type LyricsPlugin struct {
	name        string
	displayName string
	plugin      *plugin
}

// GetLyrics calls the required playback export and content-sniffs each response
// via model.ParseLyrics (TTML/SRT/YAML/LRC/plain). The response shape is the
// released contract, so older plugins and clients are unaffected.
func (l *LyricsPlugin) GetLyrics(ctx context.Context, mf *model.MediaFile) (model.LyricList, error) {
	if err := l.acquire(ctx); err != nil {
		return nil, err
	}
	defer l.release()
	return l.getLyricsV1(ctx, mf)
}

// GetLyricsDetailed prefers the optional playback-v2 export, which adds every
// cached representation and an identity per entry. A plugin without the export
// falls back to the required playback response.
func (l *LyricsPlugin) GetLyricsDetailed(ctx context.Context, mf *model.MediaFile) (model.LyricList, error) {
	if err := l.acquire(ctx); err != nil {
		return nil, err
	}
	defer l.release()
	if list, supported, err := l.getLyricsV2(ctx, mf); supported {
		return list, err
	}
	return l.getLyricsV1(ctx, mf)
}

// ListLyricsVariants returns the plugin's cached, addressable representations.
// It performs no fetch; a cold plugin cache yields an empty listing.
func (l *LyricsPlugin) ListLyricsVariants(ctx context.Context, mf *model.MediaFile) ([]model.LyricsVariant, error) {
	if err := l.acquire(ctx); err != nil {
		return nil, err
	}
	defer l.release()
	response, err := callPluginFunction[capabilities.GetLyricsVariantsRequest, capabilities.GetLyricsVariantsResponse](
		ctx, l.plugin, FuncLyricsGetVariants, capabilities.GetLyricsVariantsRequest{Track: mediaFileToTrackInfo(l.plugin, mf)},
	)
	if err != nil {
		if unsupportedPluginFunction(err) {
			return nil, nil
		}
		return nil, err
	}
	variants := make([]model.LyricsVariant, 0, len(response.Variants))
	for _, entry := range response.Variants {
		identity, ok := modelLyricsVariant(entry.Identity)
		if !ok {
			continue
		}
		selection, ok := model.EncodeLyricsSelection(l.name, identity)
		if !ok {
			continue
		}
		originalFormat := strings.ToLower(boundedText(entry.Original.Format, 16))
		original := model.LyricsOriginalInfo{
			Available:  entry.Original.Available,
			Format:     originalFormat,
			Provenance: boundedProvenance(entry.Original.Provenance),
			Reason:     boundedText(entry.Original.Reason, 160),
			Size:       max(entry.Original.Size, 0),
		}
		if original.Available && model.LyricsOriginalExtension(originalFormat) == "" {
			original.Available = false
			original.Format = ""
			if original.Reason == "" {
				original.Reason = "unsupported document format"
			}
		}
		if original.Available {
			original.MediaType = model.LyricsOriginalMediaType(originalFormat)
			original.Extension = model.LyricsOriginalExtension(originalFormat)
		}
		variants = append(variants, model.LyricsVariant{
			PluginID:      l.name,
			PluginName:    l.displayName,
			Identity:      identity,
			Selection:     selection,
			Provider:      boundedText(entry.Provider, 128),
			Language:      boundedText(entry.Language, 64),
			Role:          boundedText(entry.Role, 32),
			Format:        strings.ToLower(boundedText(entry.Format, 16)),
			PlaybackIndex: entry.PlaybackIndex,
			Original:      original,
		})
		if len(variants) >= model.MaxLyricsVariants {
			break
		}
	}
	return variants, nil
}

// GetOriginal retrieves one exact native document for one selection. The
// identity is revalidated here; a plugin-side changed, disabled or evicted
// selection is reported as model.ErrLyricsOriginalUnavailable, never replaced.
func (l *LyricsPlugin) GetOriginal(ctx context.Context, mf *model.MediaFile, identity model.LyricsVariantIdentity) (model.LyricsOriginal, error) {
	if !identity.Valid() {
		return model.LyricsOriginal{}, fmt.Errorf("%w: invalid selection", model.ErrLyricsOriginalUnavailable)
	}
	if err := l.acquire(ctx); err != nil {
		return model.LyricsOriginal{}, err
	}
	defer l.release()
	response, err := callPluginFunction[capabilities.GetLyricsOriginalRequest, capabilities.GetLyricsOriginalResponse](
		ctx, l.plugin, FuncLyricsGetOriginal,
		capabilities.GetLyricsOriginalRequest{
			Track:    mediaFileToTrackInfo(l.plugin, mf),
			Identity: capabilityLyricsVariant(identity),
		},
	)
	if err != nil {
		if unsupportedPluginFunction(err) {
			return model.LyricsOriginal{}, fmt.Errorf("%w: plugin does not provide originals", model.ErrLyricsOriginalUnavailable)
		}
		return model.LyricsOriginal{}, err
	}
	if !response.Available {
		reason := strings.TrimSpace(response.UnavailableReason)
		if reason == "" {
			reason = "selection unavailable"
		}
		return model.LyricsOriginal{}, fmt.Errorf("%w: %s", model.ErrLyricsOriginalUnavailable, reason)
	}
	returned, ok := modelLyricsVariant(response.Identity)
	if !ok || returned != identity {
		return model.LyricsOriginal{}, fmt.Errorf("%w: selection changed", model.ErrLyricsOriginalUnavailable)
	}
	format := strings.ToLower(strings.TrimSpace(response.Format))
	extension := model.LyricsOriginalExtension(format)
	mediaType := model.LyricsOriginalMediaType(format)
	if extension == "" || mediaType == "" {
		return model.LyricsOriginal{}, fmt.Errorf("%w: unsupported document format", model.ErrLyricsOriginalUnavailable)
	}
	if len(response.Document) == 0 || len(response.Document) > model.MaxLyricsOriginalBytes {
		return model.LyricsOriginal{}, fmt.Errorf("%w: document size is out of bounds", model.ErrLyricsOriginalUnavailable)
	}
	return model.LyricsOriginal{
		Identity:   identity,
		Format:     format,
		Provenance: boundedProvenance(response.Provenance),
		MediaType:  mediaType,
		Extension:  extension,
		Document:   response.Document,
	}, nil
}

func (l *LyricsPlugin) acquire(ctx context.Context) error {
	select {
	case l.plugin.lyricsSem <- struct{}{}:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (l *LyricsPlugin) release() {
	<-l.plugin.lyricsSem
}

type lyricsEntry struct {
	text    capabilities.LyricsText
	variant *capabilities.LyricsVariant
}

func (l *LyricsPlugin) getLyricsV1(ctx context.Context, mf *model.MediaFile) (model.LyricList, error) {
	response, err := callPluginFunction[capabilities.GetLyricsRequest, capabilities.GetLyricsResponse](
		ctx, l.plugin, FuncLyricsGetLyrics, capabilities.GetLyricsRequest{Track: mediaFileToTrackInfo(l.plugin, mf)},
	)
	if err != nil {
		return nil, err
	}
	entries := make([]lyricsEntry, 0, len(response.Lyrics))
	for _, text := range response.Lyrics {
		entries = append(entries, lyricsEntry{text: text})
	}
	return l.parseEntries(ctx, mf, response.Source, entries), nil
}

func (l *LyricsPlugin) getLyricsV2(ctx context.Context, mf *model.MediaFile) (model.LyricList, bool, error) {
	response, err := callPluginFunction[capabilities.GetLyricsV2Request, capabilities.GetLyricsV2Response](
		ctx, l.plugin, FuncLyricsGetLyricsV2, capabilities.GetLyricsV2Request{Track: mediaFileToTrackInfo(l.plugin, mf)},
	)
	if err != nil {
		if unsupportedPluginFunction(err) {
			return nil, false, nil
		}
		return nil, true, err
	}
	entries := make([]lyricsEntry, 0, len(response.Lyrics))
	for _, text := range response.Lyrics {
		entries = append(entries, lyricsEntry{
			text:    capabilities.LyricsText{Lang: text.Lang, Text: text.Text, Source: text.Source},
			variant: text.Variant,
		})
	}
	return l.parseEntries(ctx, mf, response.Source, entries), true, nil
}

func (l *LyricsPlugin) parseEntries(ctx context.Context, mf *model.MediaFile, responseSource *capabilities.LyricsSource, entries []lyricsEntry) model.LyricList {
	// The lyric text comes from the plugin, not the media file's own tags, so
	// attribute logs to both the plugin and the track it was fetched for.
	ctx = log.NewContext(ctx, "plugin", l.name, "file", mf.Path)
	var defaults *capabilities.LyricsSource
	if responseSource != nil {
		defaults = &capabilities.LyricsSource{
			Provider: strings.TrimSpace(responseSource.Provider),
			Format:   strings.ToLower(strings.TrimSpace(responseSource.Format)),
		}
	}

	var result model.LyricList
	for _, entry := range entries {
		lang := entry.text.Lang
		if lang == "" {
			lang = "xxx"
		}
		parsed, err := model.ParseLyrics(ctx, "", lang, []byte(entry.text.Text))
		if err != nil {
			log.Warn(ctx, "Error parsing plugin lyrics", err)
			continue
		}
		source := lyricsSourceFor(entry.text, defaults, l.displayName)
		source.PluginID = l.name
		var variant *model.LyricsVariantIdentity
		selection := ""
		if entry.variant != nil {
			if identity, ok := modelLyricsVariant(*entry.variant); ok {
				variant = &identity
				selection, _ = model.EncodeLyricsSelection(l.name, identity)
			}
		}
		for _, lyric := range parsed {
			if !lyric.IsEmpty() {
				lyric.Source = source
				lyric.Variant = variant
				lyric.Selection = selection
				result = append(result, lyric)
			}
		}
	}
	return result
}

// lyricsSourceFor resolves one entry's provenance: the entry's own source wins
// field by field, the response-level source is the default for entries without
// one, and Navidrome always supplies the plugin identity itself.
func lyricsSourceFor(entry capabilities.LyricsText, defaults *capabilities.LyricsSource, pluginName string) *model.LyricsSource {
	source := &model.LyricsSource{
		Type: model.LyricsSourcePlugin,
		Name: pluginName,
	}
	if defaults != nil {
		source.Provider = defaults.Provider
		source.Format = defaults.Format
	}
	if entry.Source != nil {
		if provider := strings.TrimSpace(entry.Source.Provider); provider != "" {
			source.Provider = provider
		}
		if format := strings.ToLower(strings.TrimSpace(entry.Source.Format)); format != "" {
			source.Format = format
		}
	}
	return source
}

func modelLyricsVariant(variant capabilities.LyricsVariant) (model.LyricsVariantIdentity, bool) {
	identity := model.LyricsVariantIdentity{
		Version:   variant.Version,
		Instance:  strings.TrimSpace(variant.Instance),
		Recording: strings.TrimSpace(variant.Recording),
		Language:  strings.TrimSpace(variant.Language),
		Role:      strings.TrimSpace(variant.Role),
		Format:    strings.ToLower(strings.TrimSpace(variant.Format)),
	}
	return identity, identity.Valid()
}

func capabilityLyricsVariant(identity model.LyricsVariantIdentity) capabilities.LyricsVariant {
	return capabilities.LyricsVariant{
		Version:   identity.Version,
		Instance:  identity.Instance,
		Recording: identity.Recording,
		Language:  identity.Language,
		Role:      identity.Role,
		Format:    identity.Format,
	}
}

func unsupportedPluginFunction(err error) bool {
	return errors.Is(err, errFunctionNotFound) || errors.Is(err, errNotImplemented)
}

func boundedText(value string, limit int) string {
	value = strings.TrimSpace(value)
	if runes := []rune(value); len(runes) > limit {
		return string(runes[:limit])
	}
	return value
}

func boundedProvenance(provenance string) string {
	switch strings.TrimSpace(provenance) {
	case "supplied", "provider-conversion", "reconstructed", "unavailable":
		return strings.TrimSpace(provenance)
	}
	return ""
}
