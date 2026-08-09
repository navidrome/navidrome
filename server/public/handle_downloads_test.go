package public

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"time"

	"github.com/navidrome/navidrome/core"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

type mockArchiver struct {
	called bool
	err    error
}

func (m *mockArchiver) ZipAlbum(context.Context, string, string, int, io.Writer) error {
	return nil
}

func (m *mockArchiver) ZipArtist(context.Context, string, string, int, io.Writer) error {
	return nil
}

func (m *mockArchiver) ZipPlaylist(context.Context, string, string, int, io.Writer) error {
	return nil
}

func (m *mockArchiver) ZipFolder(context.Context, string, string, int, io.Writer) error {
	return nil
}

func (m *mockArchiver) ZipShare(_ context.Context, _ *model.Share, w io.Writer) error {
	m.called = true
	if m.err != nil {
		return m.err
	}
	_, _ = w.Write([]byte("zip-contents"))
	return nil
}

var _ = Describe("handleDownloads", func() {
	var ds *tests.MockDataStore
	var shareRepo *tests.MockShareRepo
	var archiver *mockArchiver
	var pub *Router

	BeforeEach(func() {
		ds = &tests.MockDataStore{}
		shareRepo = &tests.MockShareRepo{}
		ds.MockedShare = shareRepo
		archiver = &mockArchiver{}
		pub = &Router{ds: ds, archiver: archiver, share: core.NewShare(ds)}
	})

	shareIs := func(s *model.Share) {
		shareRepo.ID = s.ID
		shareRepo.Entity = s
	}

	makeRequest := func(id string) *httptest.ResponseRecorder {
		r := httptest.NewRequest("GET", "/public/d/"+id+"?%3Aid="+id, nil)
		w := httptest.NewRecorder()
		pub.handleDownloads(w, r)
		return w
	}

	It("sets a Content-Disposition filename from the share description", func() {
		shareIs(&model.Share{ID: "abc123", Description: "My Mixtape", Downloadable: true})

		w := makeRequest("abc123")

		Expect(w.Code).To(Equal(http.StatusOK))
		Expect(w.Header().Get("Content-Disposition")).To(Equal(`attachment; filename="My Mixtape.zip"`))
		Expect(w.Header().Get("Content-Type")).To(Equal("application/zip"))
		Expect(archiver.called).To(BeTrue())
		Expect(w.Body.String()).To(Equal("zip-contents"))
	})

	It("falls back to the share ID when there is no description", func() {
		shareIs(&model.Share{ID: "abc123", Downloadable: true})

		w := makeRequest("abc123")

		Expect(w.Header().Get("Content-Disposition")).To(Equal(`attachment; filename="abc123.zip"`))
	})

	It("sanitizes characters that are unsafe in a filename", func() {
		shareIs(&model.Share{ID: "abc123", Description: `AC/DC: Live, 1979`, Downloadable: true})

		w := makeRequest("abc123")

		Expect(w.Header().Get("Content-Disposition")).To(Equal(`attachment; filename="AC_DC_ Live_ 1979.zip"`))
	})

	It("returns 403 without invoking the archiver when the share is not downloadable", func() {
		shareIs(&model.Share{ID: "abc123", Description: "No Download", Downloadable: false})

		w := makeRequest("abc123")

		Expect(w.Code).To(Equal(http.StatusForbidden))
		Expect(archiver.called).To(BeFalse())
		Expect(w.Header().Get("Content-Disposition")).To(BeEmpty())
	})

	It("returns 404 when the share does not exist", func() {
		shareIs(&model.Share{ID: "other", Downloadable: true})

		w := makeRequest("missing")

		Expect(w.Code).To(Equal(http.StatusNotFound))
		Expect(archiver.called).To(BeFalse())
	})

	It("returns 410 when the share has expired", func() {
		shareIs(&model.Share{ID: "abc123", Downloadable: true, ExpiresAt: new(time.Now().Add(-time.Hour))})

		w := makeRequest("abc123")

		Expect(w.Code).To(Equal(http.StatusGone))
		Expect(archiver.called).To(BeFalse())
	})

	It("returns 500 when the share lookup fails", func() {
		shareRepo.Error = errors.New("db error")

		w := makeRequest("abc123")

		Expect(w.Code).To(Equal(http.StatusInternalServerError))
		Expect(archiver.called).To(BeFalse())
	})
})
