package lyrics

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
)

const (
	maxVariantPlugins      = 8
	variantListingTimeout  = 10 * time.Second
	variantOriginalTimeout = 30 * time.Second
)

// pluginPatterns lists the configured plugin sources in priority order. Local
// sources (embedded/sidecar) have no plugin identities.
func (l *lyricsService) pluginPatterns() []string {
	patterns := make([]string, 0, maxVariantPlugins)
	seen := make(map[string]bool, maxVariantPlugins)
	for pattern := range strings.SplitSeq(conf.Server.LyricsPriority, ",") {
		pattern = strings.TrimSpace(pattern)
		if pattern == "" || isLocalSource(pattern) || seen[pattern] {
			continue
		}
		patterns = append(patterns, pattern)
		seen[pattern] = true
		if len(patterns) >= maxVariantPlugins {
			break
		}
	}
	return patterns
}

// ListLyricsVariants aggregates the addressable representations of every
// configured lyrics plugin. It is bounded and performs no fetch: playback has
// already populated the plugins' caches.
func (l *lyricsService) ListLyricsVariants(ctx context.Context, mf *model.MediaFile) ([]model.LyricsVariant, error) {
	if l.pluginLoader == nil {
		return nil, nil
	}
	ctx, cancel := context.WithTimeout(ctx, variantListingTimeout)
	defer cancel()
	variants := make([]model.LyricsVariant, 0, model.MaxLyricsVariants)
	seen := make(map[string]bool, model.MaxLyricsVariants)
	for _, pattern := range l.pluginPatterns() {
		if err := ctx.Err(); err != nil {
			return variants, err
		}
		provider, ok := l.pluginLoader.LoadLyricsProvider(pattern)
		if !ok {
			continue
		}
		originals, ok := provider.(OriginalsProvider)
		if !ok {
			continue
		}
		list, err := originals.ListLyricsVariants(ctx, mf)
		if err != nil {
			log.Debug(ctx, "Error listing lyrics variants", "plugin", pattern, err)
			continue
		}
		for _, variant := range list {
			if !variant.Identity.Valid() || variant.PluginID == "" {
				continue
			}
			key := variant.PluginID + "\x00" + variant.Identity.Key()
			if seen[key] {
				continue
			}
			seen[key] = true
			variants = append(variants, variant)
			if len(variants) >= model.MaxLyricsVariants {
				return variants, nil
			}
		}
	}
	return variants, nil
}

// GetLyricsOriginal resolves one exact selection. The plugin must be a
// currently configured lyrics source; the plugin then verifies the track, the
// enabled source and the current source instance.
func (l *lyricsService) GetLyricsOriginal(ctx context.Context, mf *model.MediaFile, plugin string, identity model.LyricsVariantIdentity) (model.LyricsOriginal, error) {
	if !identity.Valid() {
		return model.LyricsOriginal{}, fmt.Errorf("%w: invalid selection", model.ErrLyricsOriginalUnavailable)
	}
	if l.pluginLoader == nil {
		return model.LyricsOriginal{}, fmt.Errorf("%w: no plugin loader", model.ErrLyricsOriginalUnavailable)
	}
	allowed := false
	for _, pattern := range l.pluginPatterns() {
		if pattern == plugin {
			allowed = true
			break
		}
	}
	if !allowed {
		return model.LyricsOriginal{}, fmt.Errorf("%w: plugin is not a configured lyrics source", model.ErrLyricsOriginalUnavailable)
	}
	provider, ok := l.pluginLoader.LoadLyricsProvider(plugin)
	if !ok {
		return model.LyricsOriginal{}, fmt.Errorf("%w: plugin not found", model.ErrLyricsOriginalUnavailable)
	}
	originals, ok := provider.(OriginalsProvider)
	if !ok {
		return model.LyricsOriginal{}, fmt.Errorf("%w: plugin does not provide originals", model.ErrLyricsOriginalUnavailable)
	}
	ctx, cancel := context.WithTimeout(ctx, variantOriginalTimeout)
	defer cancel()
	return originals.GetOriginal(ctx, mf, identity)
}
