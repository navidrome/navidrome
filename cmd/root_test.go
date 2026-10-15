package cmd

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"path"
	"runtime/pprof"

	"github.com/go-chi/chi/v5"
	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/conf/configtest"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = pprof.NewProfile("nd-profiler-test")

var _ = Describe("profilerHandler", func() {
	// Mirrors how server.MountRouter mounts the handler.
	mount := func() http.Handler {
		router := chi.NewRouter()
		router.Mount(path.Join(conf.Server.BasePath, "/debug"), profilerHandler())
		return router
	}

	BeforeEach(func() {
		DeferCleanup(configtest.SetupConfig())
	})

	DescribeTable("serves a named profile",
		func(basePath string) {
			conf.Server.BasePath = basePath

			w := httptest.NewRecorder()
			target := path.Join(basePath, "/debug/pprof/nd-profiler-test") + "?debug=1"
			mount().ServeHTTP(w, httptest.NewRequest(http.MethodGet, target, nil))

			Expect(w.Code).To(Equal(http.StatusOK))
			Expect(w.Body.String()).To(HavePrefix("nd-profiler-test profile: total 0"))
		},
		Entry("without a BasePath", ""),
		Entry("with a BasePath", "/music"),
		Entry("with a root BasePath", "/"),
		Entry("with a trailing-slash BasePath", "/music/"),
	)
})

var _ = Describe("librariesWithChangedPID", func() {
	var ds *tests.MockDataStore
	var libs *tests.MockLibraryRepo

	BeforeEach(func() {
		DeferCleanup(configtest.SetupConfig())
		libs = &tests.MockLibraryRepo{}
		ds = &tests.MockDataStore{MockedLibrary: libs}
	})

	It("returns only the libraries whose PID config changed", func() {
		pid := model.Library{}.EffectivePID()
		libs.SetData(model.Libraries{
			{ID: 1, Name: "Same", ScannedPIDAlbum: pid.Album, ScannedPIDTrack: pid.Track},
			{ID: 2, Name: "Changed", PIDAlbum: "folder", ScannedPIDAlbum: pid.Album, ScannedPIDTrack: pid.Track},
			{ID: 3, Name: "Never scanned"},
		})
		Expect(librariesWithChangedPID(GinkgoT().Context(), ds)).To(ConsistOf("Changed", "Never scanned"))
	})

	It("returns the error from the repository", func() {
		libs.Err = errors.New("db down")
		_, err := librariesWithChangedPID(GinkgoT().Context(), ds)
		Expect(err).To(MatchError("db down"))
	})
})
