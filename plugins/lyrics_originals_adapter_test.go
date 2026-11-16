package plugins

import (
	"strings"

	"github.com/navidrome/navidrome/model"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

const (
	fixtureQRCDocument  = "<?xml version=\"1.0\" encoding=\"utf-8\"?>\n<QrcInfos>\n<QrcHeadInfo SaveTime=\"231\" Version=\"100\"/>\n<LyricInfo LyricCount=\"1\">\n<Lyric_1 LyricType=\"1\" LyricContent=\"[0,1000]line(0,500) (500,500)one\"/>\n</LyricInfo>\n</QrcInfos>\n"
	fixtureTTMLDocument = "<?xml version=\"1.0\" encoding=\"utf-8\"?>\n<tt xmlns=\"http://www.w3.org/ns/ttml\"><body><div><p begin=\"00:00.00\" end=\"00:01.00\">line one</p></div></body></tt>"
)

var _ = Describe("LyricsPlugin originals", Ordered, func() {
	var (
		originalsPlugin *LyricsPlugin
		legacyPlugin    *LyricsPlugin
	)

	BeforeAll(func() {
		manager, _ := createTestManagerWithPlugins(nil,
			"test-lyrics-originals"+PackageExtension,
			"test-lyrics"+PackageExtension,
		)
		p, ok := manager.LoadLyricsProvider("test-lyrics-originals")
		Expect(ok).To(BeTrue())
		originalsPlugin = p.(*LyricsPlugin)
		legacy, ok := manager.LoadLyricsProvider("test-lyrics")
		Expect(ok).To(BeTrue())
		legacyPlugin = legacy.(*LyricsPlugin)
	})

	track := func() *model.MediaFile {
		return &model.MediaFile{ID: "track-originals", Title: "Test Song", Artist: "Test Artist", Path: "test.mp3"}
	}

	It("keeps the required playback response for a plugin without optional exports", func() {
		list, err := legacyPlugin.GetLyrics(GinkgoT().Context(), track())
		Expect(err).ToNot(HaveOccurred())
		Expect(list).ToNot(BeEmpty())
		Expect(list[0].Variant).To(BeNil())
		Expect(list[0].Selection).To(BeEmpty())
	})

	It("falls back to the required playback response for detailed requests", func() {
		list, err := legacyPlugin.GetLyricsDetailed(GinkgoT().Context(), track())
		Expect(err).ToNot(HaveOccurred())
		Expect(list).ToNot(BeEmpty())
		Expect(list[0].Variant).To(BeNil())
	})

	It("reports unsupported optional capabilities without failing", func() {
		variants, err := legacyPlugin.ListLyricsVariants(GinkgoT().Context(), track())
		Expect(err).ToNot(HaveOccurred())
		Expect(variants).To(BeEmpty())

		identity := model.LyricsVariantIdentity{Version: 1, Instance: "native:0123456789abcdef0123456789abcdef", Recording: "rec-qrc", Role: "main", Format: "elrc"}
		_, err = legacyPlugin.GetOriginal(GinkgoT().Context(), track(), identity)
		Expect(err).To(MatchError(model.ErrLyricsOriginalUnavailable))
	})

	It("returns every representation with its identity and selection", func() {
		list, err := originalsPlugin.GetLyricsDetailed(GinkgoT().Context(), track())
		Expect(err).ToNot(HaveOccurred())
		Expect(list).To(HaveLen(2))
		Expect(list[0].Variant).ToNot(BeNil())
		Expect(list[1].Variant).ToNot(BeNil())
		Expect(list[0].Variant.Recording).To(Equal("rec-qrc"))
		Expect(list[1].Variant.Recording).To(Equal("rec-ttml"))
		Expect(list[0].Variant.Format).To(Equal("elrc"))
		Expect(list[1].Variant.Format).To(Equal("ttml"))
		Expect(list[0].Source).ToNot(BeNil())
		Expect(list[0].Source.PluginID).To(Equal("test-lyrics-originals"))
		Expect(list[0].Selection).ToNot(BeEmpty())
		plugin, identity, ok := model.DecodeLyricsSelection(list[0].Selection)
		Expect(ok).To(BeTrue())
		Expect(plugin).To(Equal("test-lyrics-originals"))
		Expect(identity).To(Equal(*list[0].Variant))
	})

	It("keeps typed evidence separate from playable entries", func() {
		result, err := originalsPlugin.GetLyricsDetailedResult(GinkgoT().Context(), track())
		Expect(err).ToNot(HaveOccurred())
		Expect(result.Lyrics).To(HaveLen(2))
		Expect(result.Lyrics[0].Match).ToNot(BeNil())
		Expect(result.Lyrics[0].Match.Status).To(Equal(model.LyricsMatchValidated))
		Expect(result.Lyrics[0].Match.AutoSave).To(BeTrue())
		Expect(result.Lyrics[1].Match).ToNot(BeNil())
		Expect(result.Lyrics[1].Match.Status).To(Equal(model.LyricsMatchAmbiguous))
		Expect(result.Lyrics[1].Match.AutoSave).To(BeFalse())
		Expect(result.Evidence).To(HaveLen(1))
		Expect(result.Evidence[0].PluginID).To(Equal("test-lyrics-originals"))
		Expect(result.Evidence[0].Kind).To(Equal("instrumental"))
	})

	It("lists the same addressable variants with matches and evidence", func() {
		variants, evidence, err := originalsPlugin.ListLyricsVariantsResult(GinkgoT().Context(), track())
		Expect(err).ToNot(HaveOccurred())
		Expect(variants).To(HaveLen(2))
		Expect(evidence).To(HaveLen(1))
		Expect(variants[0].Match).ToNot(BeNil())
		Expect(variants[0].Match.Status).To(Equal(model.LyricsMatchValidated))
		Expect(variants[1].Match.Status).To(Equal(model.LyricsMatchAmbiguous))
		Expect(variants[0].PlaybackIndex).To(Equal(0))
		Expect(variants[1].PlaybackIndex).To(Equal(1))
		Expect(variants[0].Original.Available).To(BeTrue())
		Expect(variants[0].Original.Format).To(Equal("qrc"))
		Expect(variants[1].Original.Format).To(Equal("ttml"))
		Expect(variants[0].Selection).ToNot(BeEmpty())
	})

	It("retrieves exact native documents by selection and rejects replacements", func() {
		variants, err := originalsPlugin.ListLyricsVariants(GinkgoT().Context(), track())
		Expect(err).ToNot(HaveOccurred())
		Expect(variants).To(HaveLen(2))

		qrc, err := originalsPlugin.GetOriginal(GinkgoT().Context(), track(), variants[0].Identity)
		Expect(err).ToNot(HaveOccurred())
		Expect(string(qrc.Document)).To(Equal(fixtureQRCDocument))
		Expect(qrc.MediaType).To(Equal("application/xml"))
		Expect(qrc.Extension).To(Equal(".qrc"))
		Expect(qrc.Provenance).To(Equal("supplied"))

		ttml, err := originalsPlugin.GetOriginal(GinkgoT().Context(), track(), variants[1].Identity)
		Expect(err).ToNot(HaveOccurred())
		Expect(string(ttml.Document)).To(Equal(fixtureTTMLDocument))
		Expect(ttml.MediaType).To(Equal("application/ttml+xml"))

		wrong := variants[0].Identity
		wrong.Recording = "rec-other"
		_, err = originalsPlugin.GetOriginal(GinkgoT().Context(), track(), wrong)
		Expect(err).To(MatchError(model.ErrLyricsOriginalUnavailable))

		bad := variants[0].Identity
		bad.Version = 2
		_, err = originalsPlugin.GetOriginal(GinkgoT().Context(), track(), bad)
		Expect(err).To(MatchError(model.ErrLyricsOriginalUnavailable))
	})

	It("never forwards a plugin-supplied media type", func() {
		variants, err := originalsPlugin.ListLyricsVariants(GinkgoT().Context(), track())
		Expect(err).ToNot(HaveOccurred())
		Expect(variants[0].Original.MediaType).To(Equal("application/xml"))
		Expect(strings.HasPrefix(variants[0].Original.Extension, ".")).To(BeTrue())
	})
})
