package subsonic

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"

	"github.com/go-chi/chi/v5"
	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/conf/configtest"
	"github.com/navidrome/navidrome/core/lyrics"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
	"github.com/navidrome/navidrome/server/subsonic/responses"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Lyrics personal preferences", func() {
	const trackID = "track-1"
	var router *Router
	var ds *tests.MockDataStore
	var props *tests.MockedUserPropsRepo
	var service *fakeOriginalsLyrics
	var repo *mockedMediaFile
	var mediaFile *model.MediaFile

	user := func(id string) model.User {
		return model.User{ID: id, UserName: id}
	}

	embeddedFile := func(id string) *model.MediaFile {
		list := model.LyricList{{
			Kind: model.LyricKindMain, Lang: "en", Synced: true,
			Line: []model.Line{{Value: "embedded line", Start: int64Ptr(1000)}},
		}}
		raw, err := json.Marshal(list)
		Expect(err).NotTo(HaveOccurred())
		return &model.MediaFile{
			ID: id, Title: "Song", Artist: "Artist", LibraryID: 1,
			LibraryPath: "lib", Path: "song.mp3", Lyrics: string(raw),
		}
	}

	providerVariant := func(recording, language, role, format string, match *model.LyricsMatch) model.LyricsVariant {
		identity := model.LyricsVariantIdentity{
			Version: 1, Instance: "nd-better-lyrics:0123456789abcdef0123456789abcdef",
			Recording: recording, Language: language, Role: role, Format: format,
		}
		selection, ok := model.EncodeLyricsSelection("nd-better-lyrics", identity)
		Expect(ok).To(BeTrue())
		return model.LyricsVariant{
			PluginID: "nd-better-lyrics", Provider: "LRCLib", Identity: identity,
			Selection: selection, Format: format, Language: language, Role: role, Match: match,
		}
	}

	post := func(ctx context.Context, query string) (*responses.Subsonic, error) {
		r := httptest.NewRequest(http.MethodPost, "/setLyricsPreference?"+query, nil)
		r = r.WithContext(ctx)
		return router.SetLyricsPreference(r)
	}

	readPersonal := func(ctx context.Context) (model.LyricList, *responses.LyricsPersonal) {
		locals := lyrics.LocalVariants(ctx, mediaFile)
		entries := personalEntries(service.variants, locals)
		return router.applyPersonalForRequest(ctx, mediaFile, entries, true)
	}

	BeforeEach(func() {
		DeferCleanup(configtest.SetupConfig())
		conf.Server.LyricsPriority = "embedded"
		props = &tests.MockedUserPropsRepo{}
		repo = &mockedMediaFile{MockMediaFileRepo: tests.MockMediaFileRepo{}}
		mediaFile = embeddedFile(trackID)
		repo.SetData(model.MediaFiles{*mediaFile})
		service = &fakeOriginalsLyrics{}
		ds = &tests.MockDataStore{MockedMediaFile: repo, MockedUserProps: props}
		router = New(ds, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, lyrics.NewLyrics(ds, nil), nil, nil)
		router.lyrics = service
	})

	It("durably pins and offsets one account's local variant without touching another account", func() {
		ctxA := request.WithUser(GinkgoT().Context(), user("account-a"))
		ctxB := request.WithUser(GinkgoT().Context(), user("account-b"))
		locals := lyrics.LocalVariants(ctxA, mediaFile)
		Expect(locals).To(HaveLen(1))
		token := locals[0].Token

		response, err := post(ctxA, "id="+trackID+"&action=select&local="+token+"&user=account-b")
		Expect(err).NotTo(HaveOccurred())
		Expect(response.LyricsPersonal.Selection).NotTo(BeNil())
		Expect(response.LyricsPersonal.Selection.Kind).To(Equal(model.LyricsPreferenceKindLocal))
		Expect(response.LyricsPersonal.Selection.Status).To(Equal(lyricsPersonalSelectionPinned))
		Expect(response.LyricsPersonal.Selection.Local.Token).To(Equal(token))
		Expect(response.LyricsPersonal.Selection.Selection).To(BeEmpty())

		response, err = post(ctxA, "id="+trackID+"&action=offset&local="+token+"&offsetMs=250")
		Expect(err).NotTo(HaveOccurred())
		Expect(response.LyricsPersonal.Offsets).To(HaveLen(1))
		Expect(response.LyricsPersonal.Offsets[0].OffsetMs).To(Equal(int64(250)))
		Expect(response.LyricsPersonal.Offsets[0].Local.Token).To(Equal(token))

		// The client-supplied user parameter is ignored; ownership is the
		// authenticated account.
		Expect(props.Data["account-a"+lyricsPreferenceKey(trackID)]).NotTo(BeEmpty())
		Expect(props.Data["account-b"+lyricsPreferenceKey(trackID)]).To(BeEmpty())

		// Another account sees its own empty state.
		_, personalB := readPersonal(ctxB)
		Expect(personalB.Selection).To(BeNil())
		Expect(personalB.Offsets).To(BeEmpty())

		// A fresh service/session over the same durable store still sees it.
		fresh := New(ds, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, lyrics.NewLyrics(ds, nil), nil, nil)
		fresh.lyrics = service
		entries := personalEntries(nil, lyrics.LocalVariants(ctxA, mediaFile))
		_, personalA := fresh.personalStateForTrack(ctxA, "account-a", mediaFile, entries, true)
		Expect(personalA.Selection).NotTo(BeNil())
		Expect(personalA.Selection.Status).To(Equal(lyricsPersonalSelectionPinned))
		Expect(personalA.Offsets).To(HaveLen(1))
		Expect(personalA.Offsets[0].OffsetMs).To(Equal(int64(250)))
	})

	It("retains a disappeared local choice, does not select a sibling, and recovers it", func() {
		dir := GinkgoT().TempDir()
		conf.Server.LyricsPriority = "embedded,.lrc"
		Expect(os.WriteFile(filepath.Join(dir, "song.lrc"), []byte("[00:01.00]sidecar line\n"), 0o600)).To(Succeed())
		mediaFile.LibraryPath = dir
		mediaFile.Path = "song.mp3"
		repo.SetData(model.MediaFiles{*mediaFile})
		ctx := request.WithUser(GinkgoT().Context(), user("account-a"))
		locals := lyrics.LocalVariants(ctx, mediaFile)
		Expect(locals).To(HaveLen(2))
		var sidecar model.LyricsLocalIdentity
		for _, local := range locals {
			if local.Kind == model.LyricsLocalKindSidecar {
				sidecar = local
			}
		}
		Expect(sidecar.Token).NotTo(BeEmpty())

		_, err := post(ctx, "id="+trackID+"&action=select&local="+sidecar.Token)
		Expect(err).NotTo(HaveOccurred())
		_, err = post(ctx, "id="+trackID+"&action=offset&local="+sidecar.Token+"&offsetMs=-500")
		Expect(err).NotTo(HaveOccurred())

		// Move the sidecar aside without deleting it. The embedded variant is
		// still selectable; the stored preference must not silently move to it.
		Expect(os.Rename(filepath.Join(dir, "song.lrc"), filepath.Join(dir, "song.lrc.bak"))).To(Succeed())
		_, personal := readPersonal(ctx)
		Expect(personal.Selection).NotTo(BeNil())
		Expect(personal.Selection.Status).To(Equal(lyricsPersonalSelectionGone))
		Expect(personal.Selection.Reason).To(Equal(lyricsPersonalReasonMissing))
		Expect(personal.Selection.Local.Token).To(Equal(sidecar.Token))
		Expect(personal.Selection.Selection).To(BeEmpty())
		Expect(personal.Offsets).To(BeEmpty())
		record, _, status, err := router.loadPersonal(ctx, "account-a", trackID)
		Expect(err).NotTo(HaveOccurred())
		Expect(status).To(Equal(model.LyricsPreferenceOK))
		Expect(record.Offsets).To(HaveLen(1), "the correction stays durable while its variant is unavailable")

		// The sibling is still available for automatic playback, but the
		// preference never becomes it.
		entries := personalEntries(nil, lyrics.LocalVariants(ctx, mediaFile))
		Expect(entries).To(HaveLen(1))
		Expect(entries[0].Local.Kind).To(Equal(model.LyricsLocalKindEmbedded))

		Expect(os.Rename(filepath.Join(dir, "song.lrc.bak"), filepath.Join(dir, "song.lrc"))).To(Succeed())
		_, personal = readPersonal(ctx)
		Expect(personal.Selection.Status).To(Equal(lyricsPersonalSelectionPinned))
		Expect(personal.Offsets).To(HaveLen(1))
		Expect(personal.Offsets[0].OffsetMs).To(Equal(int64(-500)))
	})

	It("rejects recordings across content revisions and keeps different recordings", func() {
		first := providerVariant("1250@0123456789abcdef0123456789abcdef", "en", "main", "lrc", &model.LyricsMatch{Status: model.LyricsMatchValidated})
		second := providerVariant("9000@0123456789abcdef0123456789abcdef", "en", "main", "lrc", &model.LyricsMatch{Status: model.LyricsMatchValidated})
		service.variants = []model.LyricsVariant{first, second}
		ctx := request.WithUser(GinkgoT().Context(), user("account-a"))

		response, err := post(ctx, "id="+trackID+"&action=reject&selection="+first.Selection)
		Expect(err).NotTo(HaveOccurred())
		Expect(response.LyricsPersonal.Rejections).To(HaveLen(1))
		rejectionToken := response.LyricsPersonal.Rejections[0].Token

		record, _, status, err := router.loadPersonal(ctx, "account-a", trackID)
		Expect(err).NotTo(HaveOccurred())
		Expect(status).To(Equal(model.LyricsPreferenceOK))

		changed := first
		changed.Identity.Recording = "1250@abcdef0123456789abcdef0123456789"
		changed.Selection, _ = model.EncodeLyricsSelection(changed.PluginID, changed.Identity)
		entries := personalEntries([]model.LyricsVariant{changed, second}, nil)
		filtered, personal := applyLyricsPersonal(record, entries, true)
		Expect(filtered).To(HaveLen(1))
		Expect(filtered[0].Variant.Recording).To(HavePrefix("9000@"))
		Expect(personal.Rejections).To(HaveLen(1))

		response, err = post(ctx, "id="+trackID+"&action=restore&rejection="+rejectionToken)
		Expect(err).NotTo(HaveOccurred())
		Expect(response.LyricsPersonal.Rejections).To(BeEmpty())
	})

	It("refuses local and content-only rejections without writing", func() {
		contentOnly := providerVariant("sha256:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef", "en", "main", "lrc", &model.LyricsMatch{Status: model.LyricsMatchValidated})
		service.variants = []model.LyricsVariant{contentOnly}
		ctx := request.WithUser(GinkgoT().Context(), user("account-a"))
		local := lyrics.LocalVariants(ctx, mediaFile)[0]

		_, err := post(ctx, "id="+trackID+"&action=reject&local="+local.Token)
		Expect(err).To(HaveOccurred())
		_, err = post(ctx, "id="+trackID+"&action=reject&selection="+contentOnly.Selection)
		Expect(err).To(HaveOccurred())
		Expect(props.Data["account-a"+lyricsPreferenceKey(trackID)]).To(BeEmpty())
	})

	It("does not overwrite an unknown persisted record version", func() {
		raw := `{"version":99,"revision":4,"selection":{"kind":"local"}}`
		props.Data = map[string]string{"account-a" + lyricsPreferenceKey(trackID): raw}
		ctx := request.WithUser(GinkgoT().Context(), user("account-a"))
		local := lyrics.LocalVariants(ctx, mediaFile)[0].Token

		_, err := post(ctx, "id="+trackID+"&action=select&local="+local)
		Expect(err).To(HaveOccurred())
		Expect(props.Data["account-a"+lyricsPreferenceKey(trackID)]).To(Equal(raw))

		_, personal := readPersonal(ctx)
		Expect(personal.Unsupported).To(BeTrue())
		Expect(personal.Selection).To(BeNil())
	})

	It("applies an earlier version record and upgrades it on the next deliberate write", func() {
		local := lyrics.LocalVariants(request.WithUser(GinkgoT().Context(), user("account-a")), mediaFile)[0]
		legacy := `{"version":0,"revision":2,"selection":{"kind":"local","local":{"version":1,"token":"` + local.Token + `","kind":"embedded","role":"main"}},"offsets":[{"kind":"local","local":{"version":1,"token":"` + local.Token + `","kind":"embedded","role":"main"},"offsetMs":100}]}`
		props.Data = map[string]string{"account-a" + lyricsPreferenceKey(trackID): legacy}
		ctx := request.WithUser(GinkgoT().Context(), user("account-a"))

		_, personal := readPersonal(ctx)
		Expect(personal.Selection).NotTo(BeNil())
		Expect(personal.Selection.Status).To(Equal(lyricsPersonalSelectionPinned))
		Expect(personal.Offsets).To(HaveLen(1))

		_, err := post(ctx, "id="+trackID+"&action=offset&local="+local.Token+"&offsetMs=0")
		Expect(err).NotTo(HaveOccurred())
		record, status := model.DecodeLyricsPreference(props.Data["account-a"+lyricsPreferenceKey(trackID)])
		Expect(status).To(Equal(model.LyricsPreferenceOK))
		Expect(record.Version).To(Equal(model.LyricsPreferenceVersion))
		Expect(record.Selection).NotTo(BeNil())
		Expect(record.Offsets).To(BeEmpty())
		Expect(record.Revision).To(Equal(int64(3)))
	})

	It("merges concurrent disjoint updates instead of losing one", func() {
		hooked := &hookedUserProps{MockedUserPropsRepo: props}
		ds.MockedUserProps = hooked
		ctx := request.WithUser(GinkgoT().Context(), user("account-a"))
		local := lyrics.LocalVariants(ctx, mediaFile)[0].Token
		hooked.before = func() {
			injected := model.LyricsPreferenceRecord{
				Version: model.LyricsPreferenceVersion, Revision: 1,
				Rejections: []model.LyricsPreferenceRejected{{
					Plugin: "nd-better-lyrics", Instance: "nd-better-lyrics:0123456789abcdef0123456789abcdef", Recording: "1250",
				}},
			}
			encoded, ok := model.EncodeLyricsPreference(injected)
			Expect(ok).To(BeTrue())
			props.Data["account-a"+lyricsPreferenceKey(trackID)] = encoded
		}

		_, err := post(ctx, "id="+trackID+"&action=offset&local="+local+"&offsetMs=250")
		Expect(err).NotTo(HaveOccurred())
		record, status := model.DecodeLyricsPreference(props.Data["account-a"+lyricsPreferenceKey(trackID)])
		Expect(status).To(Equal(model.LyricsPreferenceOK))
		Expect(record.Offsets).To(HaveLen(1))
		Expect(record.Rejections).To(HaveLen(1))
		Expect(record.Revision).To(Equal(int64(2)))
	})

	It("fails a write bounded when the store keeps conflicting", func() {
		hooked := &hookedUserProps{MockedUserPropsRepo: props}
		hooked.conflict = true
		ds.MockedUserProps = hooked
		ctx := request.WithUser(GinkgoT().Context(), user("account-a"))
		local := lyrics.LocalVariants(ctx, mediaFile)[0].Token
		_, err := post(ctx, "id="+trackID+"&action=select&local="+local)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("could not save"))
		Expect(props.Data["account-a"+lyricsPreferenceKey(trackID)]).To(BeEmpty())
	})

	It("resolves provider pins exactly, by a unique recording revision, or not at all", func() {
		first := providerVariant("1250@0123456789abcdef0123456789abcdef", "en", "main", "lrc", &model.LyricsMatch{Status: model.LyricsMatchValidated})
		identity := first.Identity
		record := model.LyricsPreferenceRecord{
			Version: model.LyricsPreferenceVersion, Revision: 1,
			Selection: &model.LyricsPreferenceSelection{
				Kind: model.LyricsPreferenceKindProvider, Plugin: first.PluginID, Provider: first.Provider, Identity: &identity,
			},
		}

		_, personal := applyLyricsPersonal(record, personalEntries([]model.LyricsVariant{first}, nil), true)
		Expect(personal.Selection.Status).To(Equal(lyricsPersonalSelectionPinned))
		Expect(personal.Selection.Selection).To(Equal(first.Selection))

		changed := first
		changed.Identity.Recording = "1250@abcdef0123456789abcdef0123456789"
		changed.Selection, _ = model.EncodeLyricsSelection(changed.PluginID, changed.Identity)
		_, personal = applyLyricsPersonal(record, personalEntries([]model.LyricsVariant{changed}, nil), true)
		Expect(personal.Selection.Status).To(Equal(lyricsPersonalSelectionRebound))
		Expect(personal.Selection.Selection).To(Equal(changed.Selection))

		siblingA := changed
		siblingA.Identity.Recording = "1250@11111111111111111111111111111111"
		siblingA.Selection, _ = model.EncodeLyricsSelection(siblingA.PluginID, siblingA.Identity)
		siblingB := changed
		siblingB.Identity.Recording = "1250@22222222222222222222222222222222"
		siblingB.Selection, _ = model.EncodeLyricsSelection(siblingB.PluginID, siblingB.Identity)
		_, personal = applyLyricsPersonal(record, personalEntries([]model.LyricsVariant{siblingA, siblingB}, nil), true)
		Expect(personal.Selection.Status).To(Equal(lyricsPersonalSelectionGone))
		Expect(personal.Selection.Reason).To(Equal(lyricsPersonalReasonAmbiguous))

		otherRecording := changed
		otherRecording.Identity.Recording = "9000@abcdef0123456789abcdef0123456789"
		otherRecording.Selection, _ = model.EncodeLyricsSelection(otherRecording.PluginID, otherRecording.Identity)
		_, personal = applyLyricsPersonal(record, personalEntries([]model.LyricsVariant{otherRecording}, nil), true)
		Expect(personal.Selection.Status).To(Equal(lyricsPersonalSelectionGone))

		_, personal = applyLyricsPersonal(record, nil, false)
		Expect(personal.Selection.Status).To(Equal(lyricsPersonalSelectionPending))
	})

	It("keeps offsets distinct by format, language and role", func() {
		base := providerVariant("1250@0123456789abcdef0123456789abcdef", "en", "main", "lrc", &model.LyricsMatch{Status: model.LyricsMatchValidated})
		elrc := base
		elrc.Identity.Format = "elrc"
		elrc.Selection, _ = model.EncodeLyricsSelection(elrc.PluginID, elrc.Identity)
		translation := base
		translation.Identity.Role = "translation"
		translation.Identity.Language = "fr"
		translation.Selection, _ = model.EncodeLyricsSelection(translation.PluginID, translation.Identity)

		identity := base.Identity
		record := model.LyricsPreferenceRecord{
			Version: model.LyricsPreferenceVersion, Revision: 1,
			Offsets: []model.LyricsPreferenceOffset{
				{Kind: model.LyricsPreferenceKindProvider, Plugin: base.PluginID, Instance: identity.Instance, Recording: "1250", Language: "en", Role: "main", Format: "lrc", Stable: true, OffsetMs: 100},
				{Kind: model.LyricsPreferenceKindProvider, Plugin: base.PluginID, Instance: identity.Instance, Recording: "1250", Language: "en", Role: "main", Format: "elrc", Stable: true, OffsetMs: 200},
				{Kind: model.LyricsPreferenceKindProvider, Plugin: base.PluginID, Instance: identity.Instance, Recording: "1250", Language: "fr", Role: "translation", Format: "lrc", Stable: true, OffsetMs: 300},
			},
		}
		_, personal := applyLyricsPersonal(record, personalEntries([]model.LyricsVariant{base, elrc, translation}, nil), true)
		Expect(personal.Offsets).To(HaveLen(3))
		bySelection := map[string]int64{}
		for _, offset := range personal.Offsets {
			bySelection[offset.Selection] = offset.OffsetMs
		}
		Expect(bySelection[base.Selection]).To(Equal(int64(100)))
		Expect(bySelection[elrc.Selection]).To(Equal(int64(200)))
		Expect(bySelection[translation.Selection]).To(Equal(int64(300)))
	})

	It("stores separate offsets for formats, languages and roles from the write path", func() {
		base := providerVariant("1250@0123456789abcdef0123456789abcdef", "en", "main", "lrc", &model.LyricsMatch{Status: model.LyricsMatchValidated})
		elrc := base
		elrc.Identity.Format = "elrc"
		elrc.Selection, _ = model.EncodeLyricsSelection(elrc.PluginID, elrc.Identity)
		translation := base
		translation.Identity.Role = "translation"
		translation.Identity.Language = "fr"
		translation.Selection, _ = model.EncodeLyricsSelection(translation.PluginID, translation.Identity)
		service.variants = []model.LyricsVariant{base, elrc, translation}
		ctx := request.WithUser(GinkgoT().Context(), user("account-a"))

		for _, variant := range []model.LyricsVariant{base, elrc, translation} {
			_, err := post(ctx, "id="+trackID+"&action=offset&selection="+variant.Selection+"&offsetMs=100")
			Expect(err).NotTo(HaveOccurred())
		}
		record, _, status, err := router.loadPersonal(ctx, "account-a", trackID)
		Expect(err).NotTo(HaveOccurred())
		Expect(status).To(Equal(model.LyricsPreferenceOK))
		Expect(record.Offsets).To(HaveLen(3))
	})

	It("requires authenticated ownership and a readable track", func() {
		local := lyrics.LocalVariants(GinkgoT().Context(), mediaFile)[0].Token
		r := httptest.NewRequest(http.MethodPost, "/setLyricsPreference?id="+trackID+"&action=select&local="+local, nil)
		_, err := router.SetLyricsPreference(r)
		Expect(err).To(HaveOccurred())

		ctx := request.WithUser(GinkgoT().Context(), user("account-a"))
		_, err = post(ctx, "id=missing&action=select&local="+local)
		Expect(err).To(HaveOccurred())
		Expect(props.Data["account-a"+lyricsPreferenceKey("missing")]).To(BeEmpty())
	})

	It("marks personalized responses private and keeps the mutation POST-only", func() {
		recorder := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/getLyricsBySongId?id="+trackID+"&enhanced=true&f=json", nil)
		_, _ = router.getLyricsBySongIdWithHeaders(recorder, r)
		Expect(recorder.Header().Get("Cache-Control")).To(Equal("private, no-store"))

		recorder = httptest.NewRecorder()
		r = httptest.NewRequest(http.MethodGet, "/getLyricsVariants?id="+trackID+"&f=json", nil)
		_, _ = router.getLyricsVariantsWithHeaders(recorder, r)
		Expect(recorder.Header().Get("Cache-Control")).To(Equal("private, no-store"))

		chiRouter := chiRouterForTest()
		postOnly(chiRouter, "setLyricsPreference", func(http.ResponseWriter, *http.Request) (*responses.Subsonic, error) {
			response := newResponse()
			return response, nil
		})
		recorder = httptest.NewRecorder()
		chiRouter.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/setLyricsPreference", nil))
		Expect(recorder.Code).To(Equal(http.StatusMethodNotAllowed))

		recorder = httptest.NewRecorder()
		chiRouter.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/setLyricsPreference", nil))
		Expect(recorder.Code).To(Equal(http.StatusOK))
		Expect(recorder.Header().Get("Cache-Control")).To(Equal("private, no-store"))
	})

	It("keeps host local identities out of exact original selections", func() {
		ctx := request.WithUser(GinkgoT().Context(), user("account-a"))
		local := lyrics.LocalVariants(ctx, mediaFile)[0]
		_, _, ok := model.DecodeLyricsSelection(local.Token)
		Expect(ok).To(BeFalse(), "a host local identity must not decode as a provider selection")
		r := httptest.NewRequest(http.MethodGet, "/getLyricsOriginal?id="+trackID+"&selection="+local.Token, nil)
		r = r.WithContext(ctx)
		recorder := httptest.NewRecorder()
		_, err := router.GetLyricsOriginal(recorder, r)
		Expect(err).To(HaveOccurred())
	})

	It("degrades to shared lyrics when durable personal state is unreadable", func() {
		props.Error = errors.New("store is down")
		ctx := request.WithUser(GinkgoT().Context(), user("account-a"))
		entries := personalEntries(service.variants, lyrics.LocalVariants(ctx, mediaFile))
		filtered, personal := router.applyPersonalForRequest(ctx, mediaFile, entries, true)
		Expect(filtered).To(HaveLen(len(entries)))
		Expect(personal).To(BeNil())
	})
})

// hookedUserProps injects a competing write between a read and its
// compare-and-swap, proving that the retry merges instead of losing state.
type hookedUserProps struct {
	*tests.MockedUserPropsRepo
	before   func()
	conflict bool
	once     sync.Once
}

func (h *hookedUserProps) CompareAndSwap(ctx context.Context, userID, key, value string, expected *string) (bool, error) {
	if h.conflict {
		return false, nil
	}
	if h.before != nil {
		h.once.Do(h.before)
	}
	return h.MockedUserPropsRepo.CompareAndSwap(ctx, userID, key, value, expected)
}

func chiRouterForTest() *chi.Mux {
	return chi.NewRouter()
}
