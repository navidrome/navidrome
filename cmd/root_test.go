package cmd

import (
	"net/http"
	"net/http/httptest"
	"path"
	"runtime/pprof"

	"github.com/go-chi/chi/v5"
	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/conf/configtest"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("profilerHandler", func() {
	// Mirrors how server.MountRouter mounts the handler.
	mount := func() http.Handler {
		router := chi.NewRouter()
		router.Mount(path.Join(conf.Server.BasePath, "/debug"), profilerHandler())
		return router
	}

	BeforeEach(func() {
		DeferCleanup(configtest.SetupConfig())
		if pprof.Lookup("nd-profiler-test") == nil {
			pprof.NewProfile("nd-profiler-test")
		}
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
	)
})
