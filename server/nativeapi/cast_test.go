package nativeapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"

	"github.com/go-chi/jwtauth/v5"
	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/conf/configtest"
	"github.com/navidrome/navidrome/core/auth"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Cast Tokens Endpoint", func() {
	var (
		ds        *tests.MockDataStore
		mfRepo    *tests.MockMediaFileRepo
		radioRepo *tests.MockedRadioRepo
		user      model.User
	)

	BeforeEach(func() {
		DeferCleanup(configtest.SetupConfig())
		conf.Server.EnableExternalServices = true
		auth.PublicTokenAuth = jwtauth.New("HS256", []byte("test-public-secret"), nil)
		auth.TokenAuth = jwtauth.New("HS256", []byte("test-session-secret"), nil)

		mfRepo = tests.CreateMockMediaFileRepo()
		mfRepo.SetData(model.MediaFiles{
			{ID: "s1", LibraryID: 1, Title: "Allowed", Suffix: "flac"},
			{ID: "s2", LibraryID: 2, Title: "Forbidden", Suffix: "mp3"},
		})
		radioRepo = tests.CreateMockedRadioRepo()
		_ = radioRepo.Put(GinkgoT().Context(), &model.Radio{ID: "r1", Name: "NTS", UploadedImage: "nts.png"})
		_ = radioRepo.Put(GinkgoT().Context(), &model.Radio{ID: "r2", Name: "Plain"})

		user = model.User{
			ID:         "u1",
			UserName:   "alice",
			TokenEpoch: 5,
			Libraries:  model.Libraries{{ID: 1}},
		}
		ds = &tests.MockDataStore{
			MockedMediaFile: mfRepo,
			MockedRadio:     radioRepo,
		}
	})

	It("mints short-lived track-scoped stream and cover tokens only for accessible libraries", func() {
		payload := castTokensRequest{
			Items: []castTokenRequestItem{
				{Key: "k1", ID: "s1", Format: "mp3", MaxBitRate: 320},
				{Key: "k2", ID: "s2", Format: "raw"},
				{Key: "r1", ID: "r1", IsRadio: true},
				{Key: "r2", ID: "r2", IsRadio: true},
			},
		}
		body, _ := json.Marshal(payload)
		req := httptest.NewRequest(http.MethodPost, "/cast/tokens", bytes.NewReader(body))
		req = req.WithContext(request.WithUser(req.Context(), user))
		w := httptest.NewRecorder()

		createCastTokens(ds)(w, req)
		Expect(w.Code).To(Equal(http.StatusOK))

		var resp castTokensResponse
		Expect(json.Unmarshal(w.Body.Bytes(), &resp)).To(Succeed())
		Expect(resp.Tokens).To(HaveKey("k1"))
		Expect(resp.Tokens).NotTo(HaveKey("k2"))
		Expect(resp.Tokens).To(HaveKey("r1"))
		Expect(resp.Tokens).NotTo(HaveKey("r2"))

		streamClaims, err := auth.ValidatePublic(resp.Tokens["k1"].StreamToken)
		Expect(err).NotTo(HaveOccurred())
		Expect(streamClaims.Audience).To(ConsistOf(auth.AudienceCast))
		Expect(streamClaims.ID).To(Equal("s1"))
		Expect(streamClaims.UserID).To(Equal("u1"))
		Expect(streamClaims.Epoch).To(Equal(5))
		Expect(streamClaims.Format).To(Equal("mp3"))
		Expect(streamClaims.BitRate).To(Equal(320))
		Expect(streamClaims.ExpiresAt.IsZero()).To(BeFalse())

		// Public cast token must never validate as a session/API token
		_, err = auth.Validate(resp.Tokens["k1"].StreamToken)
		Expect(err).To(HaveOccurred())

		coverClaims, err := auth.ValidatePublic(resp.Tokens["k1"].CoverToken)
		Expect(err).NotTo(HaveOccurred())
		Expect(coverClaims.Audience).To(BeEmpty())
		Expect(coverClaims.ID).To(HavePrefix("mf-s1"))
		Expect(coverClaims.ExpiresAt.IsZero()).To(BeFalse())

		radioCoverClaims, err := auth.ValidatePublic(resp.Tokens["r1"].CoverToken)
		Expect(err).NotTo(HaveOccurred())
		Expect(radioCoverClaims.ID).To(HavePrefix("ra-r1"))
		Expect(resp.Tokens["r1"].StreamToken).To(BeEmpty())
	})

	It("rejects unsupported formats, excessive batches, and disabled external services", func() {
		badFmt, _ := json.Marshal(castTokensRequest{
			Items: []castTokenRequestItem{{Key: "k1", ID: "s1", Format: "wav"}},
		})
		req := httptest.NewRequest(http.MethodPost, "/cast/tokens", bytes.NewReader(badFmt))
		req = req.WithContext(request.WithUser(req.Context(), user))
		w := httptest.NewRecorder()
		createCastTokens(ds)(w, req)
		Expect(w.Code).To(Equal(http.StatusBadRequest))

		tooMany := make([]castTokenRequestItem, maxCastTokenBatch+1)
		for i := range tooMany {
			tooMany[i] = castTokenRequestItem{ID: "s1", Format: "raw"}
		}
		batchBody, _ := json.Marshal(castTokensRequest{Items: tooMany})
		req2 := httptest.NewRequest(http.MethodPost, "/cast/tokens", bytes.NewReader(batchBody))
		req2 = req2.WithContext(request.WithUser(req2.Context(), user))
		w2 := httptest.NewRecorder()
		createCastTokens(ds)(w2, req2)
		Expect(w2.Code).To(Equal(http.StatusBadRequest))

		conf.Server.EnableExternalServices = false
		okBody, _ := json.Marshal(castTokensRequest{Items: []castTokenRequestItem{{ID: "s1"}}})
		req3 := httptest.NewRequest(http.MethodPost, "/cast/tokens", bytes.NewReader(okBody))
		req3 = req3.WithContext(request.WithUser(req3.Context(), user))
		w3 := httptest.NewRecorder()
		createCastTokens(ds)(w3, req3)
		Expect(w3.Code).To(Equal(http.StatusNotFound))
	})
})
