package subsonic

import (
	"context"
	"fmt"
	"net/http/httptest"
	"strings"

	"github.com/navidrome/navidrome/core/lyrics"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/server/subsonic/responses"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

type fakeOriginalsLyrics struct {
	variants     []model.LyricsVariant
	original     model.LyricsOriginal
	originalErr  error
	lastPlugin   string
	lastIdentity model.LyricsVariantIdentity
}

var _ lyrics.Lyrics = (*fakeOriginalsLyrics)(nil)

func (f *fakeOriginalsLyrics) GetLyrics(context.Context, *model.MediaFile) (model.LyricList, error) {
	return nil, nil
}

func (f *fakeOriginalsLyrics) GetLyricsVariants(context.Context, *model.MediaFile, bool) (model.LyricList, error) {
	return nil, nil
}

func (f *fakeOriginalsLyrics) GetLyricsVariantsResult(context.Context, *model.MediaFile, bool) (model.LyricsResult, error) {
	return model.LyricsResult{}, nil
}

func (f *fakeOriginalsLyrics) GetLyricsByArtistTitle(context.Context, string, string) (model.LyricList, error) {
	return nil, nil
}

func (f *fakeOriginalsLyrics) ListLyricsVariants(context.Context, *model.MediaFile) ([]model.LyricsVariant, error) {
	return f.variants, nil
}

func (f *fakeOriginalsLyrics) ListLyricsVariantsResult(context.Context, *model.MediaFile) ([]model.LyricsVariant, []model.LyricsEvidence, error) {
	return f.variants, nil, nil
}

func (f *fakeOriginalsLyrics) GetLyricsOriginal(_ context.Context, _ *model.MediaFile, plugin string, identity model.LyricsVariantIdentity) (model.LyricsOriginal, error) {
	f.lastPlugin, f.lastIdentity = plugin, identity
	return f.original, f.originalErr
}

var _ = Describe("Lyrics originals routes", func() {
	var router *Router
	var service *fakeOriginalsLyrics
	identity := model.LyricsVariantIdentity{
		Version: 1, Instance: "native:0123456789abcdef0123456789abcdef",
		Recording: "rec-qrc", Role: "main", Format: "elrc",
	}
	var selection string

	BeforeEach(func() {
		repo := &mockedMediaFile{MockMediaFileRepo: tests.MockMediaFileRepo{}}
		repo.SetData(model.MediaFiles{{
			ID: "track-1", Title: "Song/Title", Artist: "Artist", LibraryID: 1, LibraryPath: "lib",
		}})
		service = &fakeOriginalsLyrics{}
		router = New(&tests.MockDataStore{MockedMediaFile: repo}, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, service, nil, nil)
		var ok bool
		selection, ok = model.EncodeLyricsSelection("test-plugin", identity)
		Expect(ok).To(BeTrue())
	})

	It("lists addressable variants without documents", func() {
		service.variants = []model.LyricsVariant{{
			PluginID: "test-plugin", PluginName: "Test Plugin", Identity: identity, Selection: selection,
			Provider: "Native Provider", Format: "elrc", PlaybackIndex: 0,
			Original: model.LyricsOriginalInfo{Available: true, Format: "qrc", Provenance: "supplied", MediaType: "application/xml", Extension: ".qrc", Size: 42},
		}}
		request := httptest.NewRequest("GET", "/getLyricsVariants?id=track-1&f=json", nil)
		response, err := router.GetLyricsVariants(request)
		Expect(err).ToNot(HaveOccurred())
		Expect(response.LyricsVariants).ToNot(BeNil())
		Expect(response.LyricsVariants.Variants).To(HaveLen(1))
		variant := response.LyricsVariants.Variants[0]
		Expect(variant.Identity.Recording).To(Equal("rec-qrc"))
		Expect(variant.Selection).To(Equal(selection))
		Expect(variant.Provider).To(Equal("Native Provider"))
		Expect(variant.Original.Available).To(BeTrue())
		Expect(variant.Original.Format).To(Equal("qrc"))
	})

	It("returns exact bytes with trusted headers", func() {
		document := "<?xml version=\"1.0\"?><QrcInfos/>"
		service.original = model.LyricsOriginal{
			Identity: identity, Format: "qrc", Provenance: "supplied",
			MediaType: "application/xml", Extension: ".qrc", Document: []byte(document),
		}
		request := httptest.NewRequest("GET", "/getLyricsOriginal?id=track-1&selection="+selection, nil)
		recorder := httptest.NewRecorder()

		response, err := router.GetLyricsOriginal(recorder, request)
		Expect(err).ToNot(HaveOccurred())
		Expect(response).To(BeNil())
		Expect(recorder.Body.String()).To(Equal(document))
		Expect(recorder.Header().Get("Content-Type")).To(Equal("application/xml"))
		Expect(recorder.Header().Get("X-Content-Type-Options")).To(Equal("nosniff"))
		Expect(recorder.Header().Get("Content-Disposition")).To(ContainSubstring("Song_Title"))
		Expect(recorder.Header().Get("Content-Disposition")).To(ContainSubstring(".qrc"))
		Expect(service.lastPlugin).To(Equal("test-plugin"))
		Expect(service.lastIdentity).To(Equal(identity))
	})

	It("rejects invalid selections before asking for bytes", func() {
		request := httptest.NewRequest("GET", "/getLyricsOriginal?id=track-1&selection=not-a-token", nil)
		recorder := httptest.NewRecorder()
		_, err := router.GetLyricsOriginal(recorder, request)
		Expect(err).To(HaveOccurred())
		Expect(recorder.Body.String()).To(BeEmpty())
	})

	It("reports unavailable originals as data-not-found", func() {
		service.originalErr = fmt.Errorf("%w: selection no longer offered", model.ErrLyricsOriginalUnavailable)
		request := httptest.NewRequest("GET", "/getLyricsOriginal?id=track-1&selection="+selection, nil)
		recorder := httptest.NewRecorder()
		_, err := router.GetLyricsOriginal(recorder, request)
		Expect(err).To(HaveOccurred())
		subErr := mapToSubsonicError(err)
		Expect(subErr.code).To(Equal(responses.ErrorDataNotFound))
		Expect(recorder.Body.String()).To(BeEmpty())
	})

	It("requires authentication on both routes", func() {
		for _, path := range []string{
			"/getLyricsVariants?id=track-1&f=json",
			"/getLyricsOriginal?id=track-1&f=json&selection=" + selection,
		} {
			recorder := httptest.NewRecorder()
			router.ServeHTTP(recorder, httptest.NewRequest("GET", path, nil))
			Expect(recorder.Body.String()).To(ContainSubstring(`"status":"failed"`))
			Expect(recorder.Body.String()).ToNot(ContainSubstring("<QrcInfos"))
			Expect(recorder.Body.String()).ToNot(ContainSubstring("\"document\""))
		}
	})

	It("sanitizes download filenames", func() {
		name := lyricsOriginalFilename(&model.MediaFile{Title: `A/B: "C"`, Artist: "Artist"}, ".ttml")
		Expect(strings.ContainsAny(name, `/\:"`)).To(BeFalse())
		Expect(name).To(HaveSuffix(".ttml"))
	})
})
