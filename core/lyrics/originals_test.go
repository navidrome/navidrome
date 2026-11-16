package lyrics_test

import (
	"context"
	"fmt"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/conf/configtest"
	"github.com/navidrome/navidrome/core/lyrics"
	"github.com/navidrome/navidrome/model"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

type optionalLyricsProvider struct {
	standard      model.LyricList
	detailed      model.LyricList
	variants      []model.LyricsVariant
	original      model.LyricsOriginal
	err           error
	detailedCalls int
}

func (p *optionalLyricsProvider) GetLyrics(context.Context, *model.MediaFile) (model.LyricList, error) {
	return p.standard, p.err
}

func (p *optionalLyricsProvider) GetLyricsDetailed(context.Context, *model.MediaFile) (model.LyricList, error) {
	p.detailedCalls++
	return p.detailed, p.err
}

func (p *optionalLyricsProvider) ListLyricsVariants(context.Context, *model.MediaFile) ([]model.LyricsVariant, error) {
	return p.variants, p.err
}

func (p *optionalLyricsProvider) GetOriginal(context.Context, *model.MediaFile, model.LyricsVariantIdentity) (model.LyricsOriginal, error) {
	return p.original, p.err
}

type standardOnlyProvider struct {
	list model.LyricList
}

func (p *standardOnlyProvider) GetLyrics(context.Context, *model.MediaFile) (model.LyricList, error) {
	return p.list, nil
}

type staticLoader struct {
	providers map[string]lyrics.Provider
}

func (l *staticLoader) LoadLyricsProvider(name string) (lyrics.Provider, bool) {
	provider, ok := l.providers[name]
	return provider, ok
}

var _ = Describe("Lyrics variants and originals", func() {
	ctx := context.Background()
	mf := &model.MediaFile{ID: "track-1", Title: "Song", Artist: "Artist"}

	var identity model.LyricsVariantIdentity
	var selection string

	BeforeEach(func() {
		DeferCleanup(configtest.SetupConfig())
		conf.Server.LyricsPriority = "orig-plugin"
		identity = model.LyricsVariantIdentity{
			Version: 1, Instance: "plugin:0123456789abcdef0123456789abcdef",
			Recording: "rec-1", Role: "main", Format: "lrc",
		}
		var ok bool
		selection, ok = model.EncodeLyricsSelection("orig-plugin", identity)
		Expect(ok).To(BeTrue())
	})

	It("uses the detailed provider only for the enhanced variant path", func() {
		provider := &optionalLyricsProvider{
			standard: model.LyricList{{Lang: "xxx", Line: []model.Line{{Value: "standard"}}}},
			detailed: model.LyricList{{Lang: "xxx", Line: []model.Line{{Value: "detailed"}}, Variant: &identity, Selection: selection}},
		}
		svc := lyrics.NewLyrics(nil, &staticLoader{providers: map[string]lyrics.Provider{"orig-plugin": provider}})

		standard, err := svc.GetLyrics(ctx, mf)
		Expect(err).ToNot(HaveOccurred())
		Expect(standard).To(HaveLen(1))
		Expect(standard[0].Line[0].Value).To(Equal("standard"))
		Expect(provider.detailedCalls).To(BeZero())

		detailed, err := svc.GetLyricsVariants(ctx, mf, false)
		Expect(err).ToNot(HaveOccurred())
		Expect(detailed).To(HaveLen(1))
		Expect(detailed[0].Line[0].Value).To(Equal("detailed"))
		Expect(detailed[0].Variant).To(Equal(&identity))
		Expect(detailed[0].Selection).To(Equal(selection))
		Expect(provider.detailedCalls).To(Equal(1))
	})

	It("aggregates only configured plugins that expose addressable variants", func() {
		variant := model.LyricsVariant{PluginID: "orig-plugin", Identity: identity, Selection: selection}
		optional := &optionalLyricsProvider{variants: []model.LyricsVariant{variant, variant}}
		standard := &standardOnlyProvider{list: model.LyricList{{Lang: "xxx"}}}
		conf.Server.LyricsPriority = "orig-plugin,standard-only,embedded"
		svc := lyrics.NewLyrics(nil, &staticLoader{providers: map[string]lyrics.Provider{
			"orig-plugin":   optional,
			"standard-only": standard,
		}})

		variants, err := svc.ListLyricsVariants(ctx, mf)
		Expect(err).ToNot(HaveOccurred())
		Expect(variants).To(HaveLen(1))
		Expect(variants[0].Identity).To(Equal(identity))
		Expect(variants[0].Selection).To(Equal(selection))
	})

	It("bounds the global listing deterministically", func() {
		conf.Server.LyricsPriority = "orig-plugin"
		provider := &optionalLyricsProvider{}
		for index := 0; index < model.MaxLyricsVariants+5; index++ {
			candidate := identity
			candidate.Recording = fmt.Sprintf("rec-%d", index)
			selection, ok := model.EncodeLyricsSelection("orig-plugin", candidate)
			Expect(ok).To(BeTrue())
			provider.variants = append(provider.variants, model.LyricsVariant{
				PluginID: "orig-plugin", Identity: candidate, Selection: selection,
			})
		}
		svc := lyrics.NewLyrics(nil, &staticLoader{providers: map[string]lyrics.Provider{"orig-plugin": provider}})
		variants, err := svc.ListLyricsVariants(ctx, mf)
		Expect(err).ToNot(HaveOccurred())
		Expect(variants).To(HaveLen(model.MaxLyricsVariants))
		for index := range variants {
			Expect(variants[index].Identity.Recording).To(Equal(fmt.Sprintf("rec-%d", index)))
		}
	})

	It("requires the selection's plugin to be a configured lyrics source", func() {
		provider := &optionalLyricsProvider{original: model.LyricsOriginal{
			Identity: identity, Format: "lrc", MediaType: "text/plain; charset=utf-8", Extension: ".lrc", Document: []byte("line"),
		}}
		svc := lyrics.NewLyrics(nil, &staticLoader{providers: map[string]lyrics.Provider{"orig-plugin": provider}})

		original, err := svc.GetLyricsOriginal(ctx, mf, "orig-plugin", identity)
		Expect(err).ToNot(HaveOccurred())
		Expect(string(original.Document)).To(Equal("line"))

		_, err = svc.GetLyricsOriginal(ctx, mf, "other-plugin", identity)
		Expect(err).To(MatchError(model.ErrLyricsOriginalUnavailable))

		invalid := identity
		invalid.Version = 2
		_, err = svc.GetLyricsOriginal(ctx, mf, "orig-plugin", invalid)
		Expect(err).To(MatchError(model.ErrLyricsOriginalUnavailable))
	})

	It("reports unsupported plugins as unavailable without failing", func() {
		conf.Server.LyricsPriority = "standard-only"
		svc := lyrics.NewLyrics(nil, &staticLoader{providers: map[string]lyrics.Provider{
			"standard-only": &standardOnlyProvider{list: model.LyricList{{Lang: "xxx"}}},
		}})
		variants, err := svc.ListLyricsVariants(ctx, mf)
		Expect(err).ToNot(HaveOccurred())
		Expect(variants).To(BeEmpty())
		_, err = svc.GetLyricsOriginal(ctx, mf, "standard-only", identity)
		Expect(err).To(MatchError(model.ErrLyricsOriginalUnavailable))
	})
})
