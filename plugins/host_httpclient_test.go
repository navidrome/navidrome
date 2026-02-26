//go:build !windows

package plugins

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"time"

	"github.com/navidrome/navidrome/plugins/host"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("httpServiceImpl", func() {
	var (
		svc *httpServiceImpl
		ts  *httptest.Server
	)

	AfterEach(func() {
		if ts != nil {
			ts.Close()
		}
	})

	Context("without host restrictions (default SSRF protection)", func() {
		BeforeEach(func() {
			svc = newHTTPService("test-plugin", nil)
		})

		It("should block requests to loopback IPs", func() {
			ts = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(200)
			}))
			_, err := svc.Send(context.Background(), host.HTTPRequest{
				Method:    "GET",
				URL:       ts.URL,
				TimeoutMs: 1000,
			})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("private/loopback"))
		})

		It("should block requests to localhost by name", func() {
			_, err := svc.Send(context.Background(), host.HTTPRequest{
				Method:    "GET",
				URL:       "http://localhost:12345/test",
				TimeoutMs: 1000,
			})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("private/loopback"))
		})

		It("should block requests to private IPs (10.x)", func() {
			_, err := svc.Send(context.Background(), host.HTTPRequest{
				Method:    "GET",
				URL:       "http://10.0.0.1/test",
				TimeoutMs: 1000,
			})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("private/loopback"))
		})

		It("should block requests to private IPs (192.168.x)", func() {
			_, err := svc.Send(context.Background(), host.HTTPRequest{
				Method:    "GET",
				URL:       "http://192.168.1.1/test",
				TimeoutMs: 1000,
			})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("private/loopback"))
		})

		It("should block requests to private IPs (172.16.x)", func() {
			_, err := svc.Send(context.Background(), host.HTTPRequest{
				Method:    "GET",
				URL:       "http://172.16.0.1/test",
				TimeoutMs: 1000,
			})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("private/loopback"))
		})

		It("should block requests to link-local IPs (169.254.x)", func() {
			_, err := svc.Send(context.Background(), host.HTTPRequest{
				Method:    "GET",
				URL:       "http://169.254.169.254/latest/meta-data/",
				TimeoutMs: 1000,
			})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("private/loopback"))
		})

		It("should block requests to IPv6 loopback with port", func() {
			_, err := svc.Send(context.Background(), host.HTTPRequest{
				Method:    "GET",
				URL:       "http://[::1]:8080/test",
				TimeoutMs: 1000,
			})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("private/loopback"))
		})

		It("should block requests to IPv6 loopback without port", func() {
			_, err := svc.Send(context.Background(), host.HTTPRequest{
				Method:    "GET",
				URL:       "http://[::1]/test",
				TimeoutMs: 1000,
			})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("private/loopback"))
		})

		It("should allow requests to public hostnames", func() {
			// This will fail at the network level (connection refused or DNS),
			// but it should NOT fail with a "private/loopback" error
			_, err := svc.Send(context.Background(), host.HTTPRequest{
				Method:    "GET",
				URL:       "http://203.0.113.1:1/test", // TEST-NET-3, non-routable but not private
				TimeoutMs: 100,
			})
			// Should get a network error, not a permission error
			if err != nil {
				Expect(err.Error()).ToNot(ContainSubstring("private/loopback"))
			}
		})

		It("should return error for invalid URL", func() {
			_, err := svc.Send(context.Background(), host.HTTPRequest{
				Method: "GET",
				URL:    "://bad-url",
			})
			Expect(err).To(HaveOccurred())
		})

		It("should reject non-http/https URL schemes", func() {
			_, err := svc.Send(context.Background(), host.HTTPRequest{
				Method: "GET",
				URL:    "ftp://example.com/file",
			})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("must be http or https"))
		})
	})

	Context("with explicit requiredHosts allowing loopback", func() {
		BeforeEach(func() {
			svc = newHTTPService("test-plugin", &HTTPPermission{
				RequiredHosts: []string{"127.0.0.1"},
			})
		})

		It("should handle GET requests", func() {
			ts = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				Expect(r.Method).To(Equal("GET"))
				w.Header().Set("X-Test", "ok")
				w.WriteHeader(201)
				_, _ = w.Write([]byte("hello"))
			}))
			resp, err := svc.Send(context.Background(), host.HTTPRequest{
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
			resp, err := svc.Send(context.Background(), host.HTTPRequest{
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
			resp, err := svc.Send(context.Background(), host.HTTPRequest{
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
			resp, err := svc.Send(context.Background(), host.HTTPRequest{
				Method:    "DELETE",
				URL:       ts.URL,
				TimeoutMs: 1000,
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(int32(204)))
		})

		It("should handle DELETE requests with body", func() {
			ts = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				Expect(r.Method).To(Equal("DELETE"))
				b, _ := io.ReadAll(r.Body)
				_, _ = w.Write([]byte("del:" + string(b)))
			}))
			resp, err := svc.Send(context.Background(), host.HTTPRequest{
				Method:    "DELETE",
				URL:       ts.URL,
				Body:      []byte(`{"id":"123"}`),
				TimeoutMs: 1000,
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(string(resp.Body)).To(Equal(`del:{"id":"123"}`))
		})

		It("should handle PATCH requests with body", func() {
			ts = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				Expect(r.Method).To(Equal("PATCH"))
				b, _ := io.ReadAll(r.Body)
				_, _ = w.Write([]byte("patch:" + string(b)))
			}))
			resp, err := svc.Send(context.Background(), host.HTTPRequest{
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
			resp, err := svc.Send(context.Background(), host.HTTPRequest{
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
			resp, err := svc.Send(context.Background(), host.HTTPRequest{
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
			_, err := svc.Send(context.Background(), host.HTTPRequest{
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
			_, err := svc.Send(ctx, host.HTTPRequest{
				Method:    "GET",
				URL:       ts.URL,
				TimeoutMs: 5000,
			})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("context canceled"))
		})

		It("should send request headers", func() {
			ts = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				_, _ = w.Write([]byte(r.Header.Get("X-Custom")))
			}))
			resp, err := svc.Send(context.Background(), host.HTTPRequest{
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
			svc = newHTTPService("test-plugin", &HTTPPermission{
				RequiredHosts: []string{"allowed.example.com", "*.allowed.org"},
			})
		})

		It("should block requests to non-allowed hosts", func() {
			ts = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(200)
			}))
			// httptest server is on 127.0.0.1 which is not in requiredHosts
			_, err := svc.Send(context.Background(), host.HTTPRequest{
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
			resp, err := svc.Send(context.Background(), host.HTTPRequest{
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
			_, err := svc.Send(context.Background(), host.HTTPRequest{
				Method:    "GET",
				URL:       ts.URL,
				TimeoutMs: 1000,
			})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("not allowed"))
		})

		It("should block redirects to private IPs when allowlist is set", func() {
			// Server that redirects to a private IP
			ts = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				http.Redirect(w, r, "http://10.0.0.1/internal", http.StatusFound)
			}))
			// Allow the test server; redirect to 10.0.0.1 is blocked by allowlist
			svc.requiredHosts = []string{"127.0.0.1"}
			resp, err := svc.Send(context.Background(), host.HTTPRequest{
				Method:    "GET",
				URL:       ts.URL,
				TimeoutMs: 1000,
			})
			Expect(err).To(HaveOccurred())
			Expect(resp).To(BeNil())
		})

		It("should allow wildcard host patterns", func() {
			ts = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				_, _ = w.Write([]byte("wildcard"))
			}))
			// *.allowed.org is in the requiredHosts from BeforeEach, but test server is 127.0.0.1
			// Override with a wildcard that matches the test server
			svc.requiredHosts = []string{"*.0.0.1"}
			resp, err := svc.Send(context.Background(), host.HTTPRequest{
				Method:    "GET",
				URL:       ts.URL,
				TimeoutMs: 1000,
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(string(resp.Body)).To(Equal("wildcard"))
		})

		It("should reject hosts not matching wildcard patterns", func() {
			svc.requiredHosts = []string{"*.example.com"}
			_, err := svc.Send(context.Background(), host.HTTPRequest{
				Method:    "GET",
				URL:       "http://evil.other.com/test",
				TimeoutMs: 1000,
			})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("not allowed"))
		})
	})

	Context("response body size limit", func() {
		BeforeEach(func() {
			svc = newHTTPService("test-plugin", &HTTPPermission{
				RequiredHosts: []string{"127.0.0.1"},
			})
		})

		It("should truncate response body at the size limit", func() {
			// Serve a body larger than the limit
			oversizedBody := strings.Repeat("x", httpClientMaxResponseBodyLen+1024)
			ts = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				_, _ = w.Write([]byte(oversizedBody))
			}))
			resp, err := svc.Send(context.Background(), host.HTTPRequest{
				Method:    "GET",
				URL:       ts.URL,
				TimeoutMs: 5000,
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(len(resp.Body)).To(Equal(httpClientMaxResponseBodyLen))
		})
	})

	Context("edge cases", func() {
		BeforeEach(func() {
			svc = newHTTPService("test-plugin", &HTTPPermission{
				RequiredHosts: []string{"127.0.0.1"},
			})
		})

		It("should default empty method to GET", func() {
			ts = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				_, _ = w.Write([]byte("method:" + r.Method))
			}))
			// Empty method — Go's http.NewRequestWithContext normalizes "" to "GET"
			resp, err := svc.Send(context.Background(), host.HTTPRequest{
				Method:    "",
				URL:       ts.URL,
				TimeoutMs: 1000,
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(string(resp.Body)).To(Equal("method:GET"))
		})
	})
})

var _ = Describe("extractHostname", func() {
	It("should extract hostname from host:port", func() {
		Expect(extractHostname("example.com:8080")).To(Equal("example.com"))
	})

	It("should return hostname when no port", func() {
		Expect(extractHostname("example.com")).To(Equal("example.com"))
	})

	It("should handle IPv6 with port", func() {
		Expect(extractHostname("[::1]:8080")).To(Equal("::1"))
	})

	It("should handle IPv6 without port", func() {
		Expect(extractHostname("::1")).To(Equal("::1"))
	})

	It("should strip brackets from IPv6 without port", func() {
		Expect(extractHostname("[::1]")).To(Equal("::1"))
	})

	It("should handle IPv4 with port", func() {
		Expect(extractHostname("127.0.0.1:9090")).To(Equal("127.0.0.1"))
	})

	It("should handle IPv4 without port", func() {
		Expect(extractHostname("127.0.0.1")).To(Equal("127.0.0.1"))
	})
})

var _ = Describe("isPrivateOrLoopback", func() {
	It("should detect IPv4 loopback", func() {
		Expect(isPrivateOrLoopback("127.0.0.1")).To(BeTrue())
		Expect(isPrivateOrLoopback("127.0.0.2")).To(BeTrue())
	})

	It("should detect IPv6 loopback", func() {
		Expect(isPrivateOrLoopback("::1")).To(BeTrue())
	})

	It("should detect localhost by name", func() {
		Expect(isPrivateOrLoopback("localhost")).To(BeTrue())
		Expect(isPrivateOrLoopback("LOCALHOST")).To(BeTrue())
	})

	It("should detect 10.x.x.x private range", func() {
		Expect(isPrivateOrLoopback("10.0.0.1")).To(BeTrue())
		Expect(isPrivateOrLoopback("10.255.255.255")).To(BeTrue())
	})

	It("should detect 172.16.x.x private range", func() {
		Expect(isPrivateOrLoopback("172.16.0.1")).To(BeTrue())
		Expect(isPrivateOrLoopback("172.31.255.255")).To(BeTrue())
	})

	It("should detect 192.168.x.x private range", func() {
		Expect(isPrivateOrLoopback("192.168.0.1")).To(BeTrue())
		Expect(isPrivateOrLoopback("192.168.255.255")).To(BeTrue())
	})

	It("should detect link-local addresses", func() {
		Expect(isPrivateOrLoopback("169.254.169.254")).To(BeTrue())
		Expect(isPrivateOrLoopback("169.254.0.1")).To(BeTrue())
	})

	It("should detect IPv6 private (fc00::/7)", func() {
		Expect(isPrivateOrLoopback("fd00::1")).To(BeTrue())
	})

	It("should detect IPv6 link-local (fe80::/10)", func() {
		Expect(isPrivateOrLoopback("fe80::1")).To(BeTrue())
	})

	It("should allow public IPs", func() {
		Expect(isPrivateOrLoopback("8.8.8.8")).To(BeFalse())
		Expect(isPrivateOrLoopback("203.0.113.1")).To(BeFalse())
		Expect(isPrivateOrLoopback("2001:db8::1")).To(BeFalse())
	})

	It("should allow non-IP hostnames (DNS names)", func() {
		Expect(isPrivateOrLoopback("example.com")).To(BeFalse())
		Expect(isPrivateOrLoopback("api.example.com")).To(BeFalse())
	})

	It("should not treat 172.32.x.x as private", func() {
		Expect(isPrivateOrLoopback("172.32.0.1")).To(BeFalse())
	})
})
