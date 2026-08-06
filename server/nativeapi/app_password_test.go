package nativeapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// stubAppPasswordRepo gives the handler-level tests deterministic control over
// what the repo returns, without dragging in encryption / DB plumbing.
type stubAppPasswordRepo struct {
	model.AppPasswordRepository
	list      []model.AppPasswordPublic
	listErr   error
	createErr error
	deleteErr error

	createdName      string
	createdUserID    string
	createdExpiresAt *time.Time
	deletedID        string
	deletedOwnerID   string
}

func (s *stubAppPasswordRepo) ListForUser(ctx context.Context, userID string) ([]model.AppPasswordPublic, error) {
	if s.listErr != nil {
		return nil, s.listErr
	}
	out := make([]model.AppPasswordPublic, 0, len(s.list))
	for _, p := range s.list {
		if p.UserID == userID {
			out = append(out, p)
		}
	}
	return out, nil
}

func (s *stubAppPasswordRepo) Create(ctx context.Context, userID, name string, expiresAt *time.Time) (string, *model.AppPassword, error) {
	s.createdUserID = userID
	s.createdName = name
	s.createdExpiresAt = expiresAt
	if s.createErr != nil {
		return "", nil, s.createErr
	}
	ap := &model.AppPassword{
		ID:        "new-id",
		UserID:    userID,
		Name:      name,
		CreatedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		ExpiresAt: expiresAt,
	}
	return "plaintext-secret", ap, nil
}

func (s *stubAppPasswordRepo) Delete(ctx context.Context, id, ownerUserID string) error {
	s.deletedID = id
	s.deletedOwnerID = ownerUserID
	return s.deleteErr
}

var _ = Describe("App Password API", func() {
	var (
		ds     *tests.MockDataStore
		repo   *stubAppPasswordRepo
		router chi.Router
		api    *Router

		owner = model.User{ID: "user-1", UserName: "alice", IsAdmin: false}
		admin = model.User{ID: "admin-1", UserName: "root", IsAdmin: true}
		other = model.User{ID: "user-2", UserName: "bob", IsAdmin: false}
	)

	BeforeEach(func() {
		repo = &stubAppPasswordRepo{}
		ds = &tests.MockDataStore{MockedAppPassword: repo}
		api = &Router{ds: ds}

		router = chi.NewRouter()
		api.addAppPasswordRoute(router)
	})

	// serve wraps the user into the request context (the way JWTVerifier would
	// in production) and exercises the chi router so the URL params are
	// populated for the middleware and handlers.
	serve := func(method, path string, body []byte, caller *model.User) *httptest.ResponseRecorder {
		var reader *bytes.Reader
		if body != nil {
			reader = bytes.NewReader(body)
		} else {
			reader = bytes.NewReader(nil)
		}
		req := httptest.NewRequest(method, path, reader)
		req.Header.Set("Content-Type", "application/json")
		if caller != nil {
			req = req.WithContext(request.WithUser(req.Context(), *caller))
		}
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		return w
	}

	Describe("appPasswordOwnerOrAdminMiddleware", func() {
		It("lets the owner manage their own passwords", func() {
			w := serve("GET", "/user/user-1/app-password/", nil, &owner)
			Expect(w.Code).To(Equal(http.StatusOK))
		})

		It("lets an admin manage any user's passwords", func() {
			w := serve("GET", "/user/user-1/app-password/", nil, &admin)
			Expect(w.Code).To(Equal(http.StatusOK))
		})

		It("forbids non-owner non-admin access", func() {
			w := serve("GET", "/user/user-1/app-password/", nil, &other)
			Expect(w.Code).To(Equal(http.StatusForbidden))
		})

		It("rejects unauthenticated requests with 401", func() {
			w := serve("GET", "/user/user-1/app-password/", nil, nil)
			Expect(w.Code).To(Equal(http.StatusUnauthorized))
		})
	})

	Describe("listAppPasswords", func() {
		It("returns the public projection without any secret fields", func() {
			repo.list = []model.AppPasswordPublic{
				{ID: "ap-1", UserID: "user-1", Name: "Phone", CreatedAt: time.Now()},
				{ID: "ap-2", UserID: "user-1", Name: "Laptop", CreatedAt: time.Now()},
			}
			w := serve("GET", "/user/user-1/app-password/", nil, &owner)
			Expect(w.Code).To(Equal(http.StatusOK))

			// Decode into the typed projection - if any unexpected secret field
			// leaks in, json.Decoder won't see it here, so additionally inspect
			// the raw bytes.
			var got []model.AppPasswordPublic
			Expect(json.Unmarshal(w.Body.Bytes(), &got)).To(Succeed())
			Expect(got).To(HaveLen(2))

			raw := w.Body.String()
			Expect(raw).NotTo(ContainSubstring("secret"))
			Expect(raw).NotTo(ContainSubstring("Secret"))
			Expect(raw).NotTo(ContainSubstring("secret_encrypted"))
		})

		It("returns 500 when the repo errors", func() {
			repo.listErr = errors.New("boom")
			w := serve("GET", "/user/user-1/app-password/", nil, &owner)
			Expect(w.Code).To(Equal(http.StatusInternalServerError))
		})
	})

	Describe("createAppPassword", func() {
		It("returns 201 with the one-time plaintext secret", func() {
			body, _ := json.Marshal(map[string]any{"name": "Phone"})
			w := serve("POST", "/user/user-1/app-password/", body, &owner)
			Expect(w.Code).To(Equal(http.StatusCreated))

			var resp createAppPasswordResponse
			Expect(json.Unmarshal(w.Body.Bytes(), &resp)).To(Succeed())
			Expect(resp.ID).To(Equal("new-id"))
			Expect(resp.Name).To(Equal("Phone"))
			Expect(resp.Secret).To(Equal("plaintext-secret"))

			Expect(repo.createdUserID).To(Equal("user-1"))
			Expect(repo.createdName).To(Equal("Phone"))
			Expect(repo.createdExpiresAt).To(BeNil())
		})

		It("forwards expiresAt to the repo", func() {
			exp := time.Date(2030, 1, 2, 3, 4, 5, 0, time.UTC)
			body, _ := json.Marshal(map[string]any{"name": "TempKey", "expiresAt": exp})
			w := serve("POST", "/user/user-1/app-password/", body, &owner)
			Expect(w.Code).To(Equal(http.StatusCreated))
			Expect(repo.createdExpiresAt).NotTo(BeNil())
			Expect(repo.createdExpiresAt.Equal(exp)).To(BeTrue())
		})

		It("rejects an empty name with 422", func() {
			body, _ := json.Marshal(map[string]any{"name": ""})
			w := serve("POST", "/user/user-1/app-password/", body, &owner)
			Expect(w.Code).To(Equal(http.StatusUnprocessableEntity))
			Expect(repo.createdName).To(Equal("")) // repo was not called
			Expect(repo.createdUserID).To(Equal(""))
		})

		It("rejects a malformed body with 422", func() {
			w := serve("POST", "/user/user-1/app-password/", []byte("{not json"), &owner)
			Expect(w.Code).To(Equal(http.StatusUnprocessableEntity))
		})

		It("rejects access from a non-owner before reaching the handler", func() {
			body, _ := json.Marshal(map[string]any{"name": "Phone"})
			w := serve("POST", "/user/user-1/app-password/", body, &other)
			Expect(w.Code).To(Equal(http.StatusForbidden))
			Expect(repo.createdName).To(Equal("")) // repo was not called
		})

		It("returns 500 when the repo errors", func() {
			repo.createErr = errors.New("boom")
			body, _ := json.Marshal(map[string]any{"name": "Phone"})
			w := serve("POST", "/user/user-1/app-password/", body, &owner)
			Expect(w.Code).To(Equal(http.StatusInternalServerError))
		})
	})

	Describe("deleteAppPassword", func() {
		It("returns 204 on success and forwards the owner scope", func() {
			w := serve("DELETE", "/user/user-1/app-password/ap-1", nil, &owner)
			Expect(w.Code).To(Equal(http.StatusNoContent))
			Expect(repo.deletedID).To(Equal("ap-1"))
			Expect(repo.deletedOwnerID).To(Equal("user-1"))
		})

		It("returns 404 when the repo reports not found", func() {
			repo.deleteErr = model.ErrNotFound
			w := serve("DELETE", "/user/user-1/app-password/missing", nil, &owner)
			Expect(w.Code).To(Equal(http.StatusNotFound))
		})

		It("returns 500 on unexpected repo errors", func() {
			repo.deleteErr = errors.New("boom")
			w := serve("DELETE", "/user/user-1/app-password/ap-1", nil, &owner)
			Expect(w.Code).To(Equal(http.StatusInternalServerError))
		})

		It("forbids non-owner non-admin callers before reaching the repo", func() {
			w := serve("DELETE", "/user/user-1/app-password/ap-1", nil, &other)
			Expect(w.Code).To(Equal(http.StatusForbidden))
			Expect(repo.deletedID).To(Equal(""))
		})

		It("scopes admin deletes to the path's userId, not the admin's", func() {
			// Even when an admin deletes on behalf of someone else, the
			// owner-id passed to the repo must be the path's userId so the
			// WHERE clause stays correct.
			w := serve("DELETE", "/user/user-1/app-password/ap-1", nil, &admin)
			Expect(w.Code).To(Equal(http.StatusNoContent))
			Expect(repo.deletedOwnerID).To(Equal("user-1"))
		})
	})

	Describe("end-to-end URL shapes", func() {
		It("404s for an unknown subpath under app-password", func() {
			w := serve("GET", "/user/user-1/app-password/unknown/extra", nil, &owner)
			Expect(w.Code).To(Equal(http.StatusNotFound))
			// nothing should hit the create/delete tracking
			Expect(repo.deletedID).To(Equal(""))
		})

		It("preserves the URL parameter through the middleware (sanity)", func() {
			// Smoke-test: hit a path that includes a hyphen in the userId,
			// confirm the middleware sees the same value chi parsed.
			hyphenated := model.User{ID: "user-with-hyphen", UserName: "x", IsAdmin: false}
			w := serve("GET", "/user/user-with-hyphen/app-password/", nil, &hyphenated)
			Expect(w.Code).To(Equal(http.StatusOK))
			// The list path uses URLParam internally - we hit the trailing
			// slash one so chi resolves the param.
			Expect(strings.Contains(w.Body.String(), "[]")).To(BeTrue())
		})
	})
})
