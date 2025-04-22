package plugins

import (
	"context"
	"net/http"
	"net/http/httptest"
	"time"

	hosthttp "github.com/navidrome/navidrome/plugins/host"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("HttpService", func() {
	var (
		svc *HttpService
		ts  *httptest.Server
	)

	BeforeEach(func() {
		svc = &HttpService{}
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
			w.Write([]byte("hello"))
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
			r.Body.Read(b)
			w.Write([]byte("got:" + string(b)))
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
			r.Body.Read(b)
			w.Write([]byte("put:" + string(b)))
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

	It("should handle timeouts and errors", func() {
		ts = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(50 * time.Millisecond)
		}))
		resp, err := svc.Get(context.Background(), &hosthttp.HttpRequest{
			Url:       ts.URL,
			TimeoutMs: 1,
		})
		Expect(err).To(BeNil())
		Expect(resp.Error).NotTo(BeEmpty())
	})
})
