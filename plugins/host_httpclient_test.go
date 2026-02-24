//go:build !windows

package plugins

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"time"

	"github.com/navidrome/navidrome/plugins/host"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("httpClientServiceImpl", func() {
	var (
		svc *httpClientServiceImpl
		ts  *httptest.Server
	)

	AfterEach(func() {
		if ts != nil {
			ts.Close()
		}
	})

	Context("without host restrictions", func() {
		BeforeEach(func() {
			svc = &httpClientServiceImpl{
				pluginName: "test-plugin",
			}
		})

		It("should handle GET requests", func() {
			ts = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				Expect(r.Method).To(Equal("GET"))
				w.Header().Set("X-Test", "ok")
				w.WriteHeader(201)
				_, _ = w.Write([]byte("hello"))
			}))
			resp, err := svc.Do(context.Background(), host.HttpRequest{
				Method:    "GET",
				URL:       ts.URL,
				Headers:   map[string]string{"Accept": "text/plain"},
				TimeoutMs: 1000,
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(int32(201)))
			Expect(string(resp.Body)).To(Equal("hello"))
			Expect(resp.Headers["X-Test"]).To(Equal("ok"))
		})

		It("should handle POST requests with body", func() {
			ts = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				Expect(r.Method).To(Equal("POST"))
				b, _ := io.ReadAll(r.Body)
				_, _ = w.Write([]byte("got:" + string(b)))
			}))
			resp, err := svc.Do(context.Background(), host.HttpRequest{
				Method:    "POST",
				URL:       ts.URL,
				Body:      []byte("abc"),
				TimeoutMs: 1000,
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(string(resp.Body)).To(Equal("got:abc"))
		})

		It("should handle PUT requests with body", func() {
			ts = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				Expect(r.Method).To(Equal("PUT"))
				b, _ := io.ReadAll(r.Body)
				_, _ = w.Write([]byte("put:" + string(b)))
			}))
			resp, err := svc.Do(context.Background(), host.HttpRequest{
				Method:    "PUT",
				URL:       ts.URL,
				Body:      []byte("xyz"),
				TimeoutMs: 1000,
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(string(resp.Body)).To(Equal("put:xyz"))
		})

		It("should handle DELETE requests", func() {
			ts = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				Expect(r.Method).To(Equal("DELETE"))
				w.WriteHeader(204)
			}))
			resp, err := svc.Do(context.Background(), host.HttpRequest{
				Method:    "DELETE",
				URL:       ts.URL,
				TimeoutMs: 1000,
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(int32(204)))
		})

		It("should handle PATCH requests with body", func() {
			ts = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				Expect(r.Method).To(Equal("PATCH"))
				b, _ := io.ReadAll(r.Body)
				_, _ = w.Write([]byte("patch:" + string(b)))
			}))
			resp, err := svc.Do(context.Background(), host.HttpRequest{
				Method:    "PATCH",
				URL:       ts.URL,
				Body:      []byte("data"),
				TimeoutMs: 1000,
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(string(resp.Body)).To(Equal("patch:data"))
		})

		It("should handle HEAD requests", func() {
			ts = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				Expect(r.Method).To(Equal("HEAD"))
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(200)
			}))
			resp, err := svc.Do(context.Background(), host.HttpRequest{
				Method:    "HEAD",
				URL:       ts.URL,
				TimeoutMs: 1000,
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(int32(200)))
			Expect(resp.Headers["Content-Type"]).To(Equal("application/json"))
			Expect(resp.Body).To(BeEmpty())
		})

		It("should use default timeout when TimeoutMs is 0", func() {
			ts = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(200)
			}))
			resp, err := svc.Do(context.Background(), host.HttpRequest{
				Method: "GET",
				URL:    ts.URL,
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(int32(200)))
		})

		It("should return error on timeout", func() {
			ts = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				time.Sleep(50 * time.Millisecond)
			}))
			_, err := svc.Do(context.Background(), host.HttpRequest{
				Method:    "GET",
				URL:       ts.URL,
				TimeoutMs: 1,
			})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("deadline exceeded"))
		})

		It("should return error on context cancellation", func() {
			ts = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				time.Sleep(50 * time.Millisecond)
			}))
			ctx, cancel := context.WithCancel(context.Background())
			go func() {
				time.Sleep(1 * time.Millisecond)
				cancel()
			}()
			_, err := svc.Do(ctx, host.HttpRequest{
				Method:    "GET",
				URL:       ts.URL,
				TimeoutMs: 5000,
			})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("context canceled"))
		})

		It("should return error for invalid URL", func() {
			_, err := svc.Do(context.Background(), host.HttpRequest{
				Method: "GET",
				URL:    "://bad-url",
			})
			Expect(err).To(HaveOccurred())
		})

		It("should reject non-http/https URL schemes", func() {
			_, err := svc.Do(context.Background(), host.HttpRequest{
				Method: "GET",
				URL:    "ftp://example.com/file",
			})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("must be http or https"))
		})

		It("should send request headers", func() {
			ts = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				_, _ = w.Write([]byte(r.Header.Get("X-Custom")))
			}))
			resp, err := svc.Do(context.Background(), host.HttpRequest{
				Method:    "GET",
				URL:       ts.URL,
				Headers:   map[string]string{"X-Custom": "myvalue"},
				TimeoutMs: 1000,
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(string(resp.Body)).To(Equal("myvalue"))
		})
	})

	Context("with host restrictions", func() {
		BeforeEach(func() {
			svc = &httpClientServiceImpl{
				pluginName:    "test-plugin",
				requiredHosts: []string{"allowed.example.com", "*.allowed.org"},
			}
		})

		It("should block requests to non-allowed hosts", func() {
			ts = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(200)
			}))
			// httptest server is on 127.0.0.1 which is not in requiredHosts
			_, err := svc.Do(context.Background(), host.HttpRequest{
				Method:    "GET",
				URL:       ts.URL,
				TimeoutMs: 1000,
			})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("not allowed"))
		})

		It("should follow redirects to allowed hosts", func() {
			// Create a destination server
			dest := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				_, _ = w.Write([]byte("final"))
			}))
			defer dest.Close()
			// Create a redirect server
			ts = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				http.Redirect(w, r, dest.URL, http.StatusFound)
			}))
			// Allow both servers (both on 127.0.0.1)
			svc.requiredHosts = []string{"127.0.0.1"}
			resp, err := svc.Do(context.Background(), host.HttpRequest{
				Method:    "GET",
				URL:       ts.URL,
				TimeoutMs: 1000,
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(int32(200)))
			Expect(string(resp.Body)).To(Equal("final"))
		})

		It("should block redirects to non-allowed hosts", func() {
			// Server that redirects to a disallowed host
			ts = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				http.Redirect(w, r, "http://evil.example.com/steal", http.StatusFound)
			}))
			// Override requiredHosts to allow the test server
			svc.requiredHosts = []string{"127.0.0.1"}
			_, err := svc.Do(context.Background(), host.HttpRequest{
				Method:    "GET",
				URL:       ts.URL,
				TimeoutMs: 1000,
			})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("not allowed"))
		})
	})
})
