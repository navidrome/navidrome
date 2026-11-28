package cmd

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"path"
	"runtime/pprof"

	"github.com/go-chi/chi/v5"
	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/db"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/persistence"
	"github.com/navidrome/navidrome/persistence/persistencetest"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = pprof.NewProfile("nd-profiler-test")

var _ = Describe("profilerHandler", func() {
	var ctx context.Context
	BeforeEach(func() {
		ctx = GinkgoT().Context()
	})

	// Mirrors how server.MountRouter mounts the handler.
	mount := func() http.Handler {
		router := chi.NewRouter()
		router.Mount(path.Join(conf.Server().BasePath, "/debug"), profilerHandler())
		return router
	}

	DescribeTable("serves a named profile",
		func(basePath string) {
			conf.Update(func(c *conf.Options) { c.BasePath = basePath })

			w := httptest.NewRecorder()
			target := path.Join(basePath, "/debug/pprof/nd-profiler-test") + "?debug=1"
			mount().ServeHTTP(w, httptest.NewRequestWithContext(ctx, http.MethodGet, target, nil))

			Expect(w.Code).To(Equal(http.StatusOK))
			Expect(w.Body.String()).To(HavePrefix("nd-profiler-test profile: total 0"))
		},
		Entry("without a BasePath", ""),
		Entry("with a BasePath", "/music"),
		Entry("with a root BasePath", "/"),
		Entry("with a trailing-slash BasePath", "/music/"),
	)
})

type failingLibraryRepo struct {
	model.LibraryRepository
	err error
}

func (r failingLibraryRepo) GetAll(context.Context, ...model.QueryOptions) (model.Libraries, error) {
	return nil, r.err
}

var _ = Describe("librariesWithChangedPID", func() {
	var ctx context.Context
	var ds model.DataStore

	BeforeEach(func() {
		ctx = GinkgoT().Context()
		ds = persistence.New(db.Db())
	})

	It("returns only the libraries whose PID config changed", func() {
		pid := model.Library{}.EffectivePID()
		persistencetest.SetLibraries(ctx, ds, "/music/same", "/music/changed", "/music/never")
		same, err := ds.Library().Get(ctx, 1)
		Expect(err).ToNot(HaveOccurred())
		same.Name = "Same"
		Expect(ds.Library().Put(ctx, same)).To(Succeed())
		Expect(ds.Library().Put(ctx, &model.Library{ID: 2, Name: "Changed", Path: "/music/changed", PIDAlbum: "folder"})).To(Succeed())
		Expect(ds.Library().Put(ctx, &model.Library{ID: 3, Name: "Never scanned", Path: "/music/never"})).To(Succeed())
		Expect(ds.Library().SetScannedPID(ctx, 1, pid)).To(Succeed())
		Expect(ds.Library().SetScannedPID(ctx, 2, pid)).To(Succeed())

		Expect(librariesWithChangedPID(ctx, ds)).To(ConsistOf("Changed", "Never scanned"))
	})

	It("returns the error from the repository", func() {
		ds = persistencetest.Stub(ds, persistencetest.Repos{Library: func(r model.LibraryRepository) model.LibraryRepository {
			return failingLibraryRepo{LibraryRepository: r, err: errors.New("db down")}
		}})
		_, err := librariesWithChangedPID(ctx, ds)
		Expect(err).To(MatchError("db down"))
	})
})
