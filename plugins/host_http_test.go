package plugins

import (
	"context"
	"net/http"
	"net/http/httptest"
	"time"

	hosthttp "github.com/navidrome/navidrome/plugins/host/http"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("httpServiceImpl", func() {
	var (
		svc *httpServiceImpl
		ts  *httptest.Server
	)

	BeforeEach(func() {
		svc = &httpServiceImpl{}
	})

	AfterEach(func() {
		if ts != nil {
			ts.Close()
		}
	})

	It("should handle GET requests", func() {
		ts = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Test", "ok")
			w.WriteHeader(201)
			_, _ = w.Write([]byte("hello"))
		}))
		resp, err := svc.Get(context.Background(), &hosthttp.HttpRequest{
			Url:       ts.URL,
			Headers:   map[string]string{"A": "B"},
			TimeoutMs: 1000,
		})
		Expect(err).To(BeNil())
		Expect(resp.Error).To(BeEmpty())
		Expect(resp.Status).To(Equal(int32(201)))
		Expect(string(resp.Body)).To(Equal("hello"))
		Expect(resp.Headers["X-Test"]).To(Equal("ok"))
	})

	It("should handle POST requests with body", func() {
		ts = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			b := make([]byte, r.ContentLength)
			_, _ = r.Body.Read(b)
			_, _ = w.Write([]byte("got:" + string(b)))
		}))
		resp, err := svc.Post(context.Background(), &hosthttp.HttpRequest{
			Url:       ts.URL,
			Body:      []byte("abc"),
			TimeoutMs: 1000,
		})
		Expect(err).To(BeNil())
		Expect(resp.Error).To(BeEmpty())
		Expect(string(resp.Body)).To(Equal("got:abc"))
	})

	It("should handle PUT requests with body", func() {
		ts = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			b := make([]byte, r.ContentLength)
			_, _ = r.Body.Read(b)
			_, _ = w.Write([]byte("put:" + string(b)))
		}))
		resp, err := svc.Put(context.Background(), &hosthttp.HttpRequest{
			Url:       ts.URL,
			Body:      []byte("xyz"),
			TimeoutMs: 1000,
		})
		Expect(err).To(BeNil())
		Expect(resp.Error).To(BeEmpty())
		Expect(string(resp.Body)).To(Equal("put:xyz"))
	})

	It("should handle DELETE requests", func() {
		ts = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(204)
		}))
		resp, err := svc.Delete(context.Background(), &hosthttp.HttpRequest{
			Url:       ts.URL,
			TimeoutMs: 1000,
		})
		Expect(err).To(BeNil())
		Expect(resp.Error).To(BeEmpty())
		Expect(resp.Status).To(Equal(int32(204)))
	})

	It("should handle PATCH requests with body", func() {
		ts = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			b := make([]byte, r.ContentLength)
			_, _ = r.Body.Read(b)
			_, _ = w.Write([]byte("patch:" + string(b)))
		}))
		resp, err := svc.Patch(context.Background(), &hosthttp.HttpRequest{
			Url:       ts.URL,
			Body:      []byte("test-patch"),
			TimeoutMs: 1000,
		})
		Expect(err).To(BeNil())
		Expect(resp.Error).To(BeEmpty())
		Expect(string(resp.Body)).To(Equal("patch:test-patch"))
	})

	It("should handle HEAD requests", func() {
		ts = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Content-Length", "42")
			w.WriteHeader(200)
			// HEAD responses shouldn't have a body, but the headers should be present
		}))
		resp, err := svc.Head(context.Background(), &hosthttp.HttpRequest{
			Url:       ts.URL,
			TimeoutMs: 1000,
		})
		Expect(err).To(BeNil())
		Expect(resp.Error).To(BeEmpty())
		Expect(resp.Status).To(Equal(int32(200)))
		Expect(resp.Headers["Content-Type"]).To(Equal("application/json"))
		Expect(resp.Headers["Content-Length"]).To(Equal("42"))
		Expect(resp.Body).To(BeEmpty()) // HEAD responses have no body
	})

	It("should handle OPTIONS requests", func() {
		ts = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Allow", "GET, POST, PUT, DELETE, PATCH, HEAD, OPTIONS")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, PATCH, HEAD, OPTIONS")
			w.WriteHeader(200)
		}))
		resp, err := svc.Options(context.Background(), &hosthttp.HttpRequest{
			Url:       ts.URL,
			TimeoutMs: 1000,
		})
		Expect(err).To(BeNil())
		Expect(resp.Error).To(BeEmpty())
		Expect(resp.Status).To(Equal(int32(200)))
		Expect(resp.Headers["Allow"]).To(Equal("GET, POST, PUT, DELETE, PATCH, HEAD, OPTIONS"))
		Expect(resp.Headers["Access-Control-Allow-Methods"]).To(Equal("GET, POST, PUT, DELETE, PATCH, HEAD, OPTIONS"))
	})

	It("should handle timeouts and errors", func() {
		ts = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(50 * time.Millisecond)
		}))
		resp, err := svc.Get(context.Background(), &hosthttp.HttpRequest{
			Url:       ts.URL,
			TimeoutMs: 1,
		})
		Expect(err).To(BeNil())
		Expect(resp).NotTo(BeNil())
		Expect(resp.Error).To(ContainSubstring("deadline exceeded"))
	})

	It("should return error on context timeout", func() {
		ts = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(50 * time.Millisecond)
		}))
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
		defer cancel()
		resp, err := svc.Get(ctx, &hosthttp.HttpRequest{
			Url:       ts.URL,
			TimeoutMs: 1000,
		})
		Expect(err).To(BeNil())
		Expect(resp).NotTo(BeNil())
		Expect(resp.Error).To(ContainSubstring("context deadline exceeded"))
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
		resp, err := svc.Get(ctx, &hosthttp.HttpRequest{
			Url:       ts.URL,
			TimeoutMs: 1000,
		})
		Expect(err).To(BeNil())
		Expect(resp).NotTo(BeNil())
		Expect(resp.Error).To(ContainSubstring("context canceled"))
	})
})
