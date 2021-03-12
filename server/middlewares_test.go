package server

import (
	"net/http"
	"net/http/httptest"
	"os"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("middlewares", func() {
	var nextCalled bool
	next := func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
	}
	Describe("robotsTXT", func() {
		BeforeEach(func() {
			nextCalled = false
		})

		It("returns the robot.txt when requested from root", func() {
			r := httptest.NewRequest("GET", "/robots.txt", nil)
			w := httptest.NewRecorder()

			robotsTXT(os.DirFS("tests/fixtures"))(http.HandlerFunc(next)).ServeHTTP(w, r)

			Expect(nextCalled).To(BeFalse())
			Expect(w.Body.String()).To(HavePrefix("User-agent:"))
		})

		It("allows prefixes", func() {
			r := httptest.NewRequest("GET", "/app/robots.txt", nil)
			w := httptest.NewRecorder()

			robotsTXT(os.DirFS("tests/fixtures"))(http.HandlerFunc(next)).ServeHTTP(w, r)

			Expect(nextCalled).To(BeFalse())
			Expect(w.Body.String()).To(HavePrefix("User-agent:"))
		})

		It("passes through requests for other files", func() {
			r := httptest.NewRequest("GET", "/this_is_not_a_robots.txt_file", nil)
			w := httptest.NewRecorder()

			robotsTXT(os.DirFS("tests/fixtures"))(http.HandlerFunc(next)).ServeHTTP(w, r)

			Expect(nextCalled).To(BeTrue())
		})
	})
})
