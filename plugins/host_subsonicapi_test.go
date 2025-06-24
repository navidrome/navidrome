package plugins

import (
	"context"
	"net/http"
	"sync"

	"github.com/navidrome/navidrome/model/request"
	"github.com/navidrome/navidrome/plugins/host/subsonicapi"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("SubsonicAPI Host Service", func() {
	var (
		service    *subsonicAPIServiceImpl
		manager    *Manager
		mockRouter http.Handler
	)

	BeforeEach(func() {
		// Create a mock router
		mockRouter = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"subsonic-response":{"status":"ok","version":"1.16.1"}}`))
		})

		// Create manager with mock router
		manager = &Manager{
			mu:             sync.RWMutex{},
			subsonicRouter: mockRouter,
		}

		// Create service implementation
		service = &subsonicAPIServiceImpl{
			pluginID: "test-plugin",
			router:   manager.subsonicRouter,
		}
	})

	Describe("Call", func() {
		Context("when subsonic router is available", func() {
			It("should process the request successfully", func() {
				req := &subsonicapi.CallRequest{
					Url: "/rest/ping?u=admin",
				}

				resp, err := service.Call(context.Background(), req)

				Expect(err).ToNot(HaveOccurred())
				Expect(resp).ToNot(BeNil())
				Expect(resp.Error).To(BeEmpty())
				Expect(resp.Json).To(ContainSubstring("subsonic-response"))
				Expect(resp.Json).To(ContainSubstring("ok"))
			})

			It("should add required parameters to the URL", func() {
				var capturedRequest *http.Request
				mockRouter = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					capturedRequest = r
					w.WriteHeader(http.StatusOK)
					_, _ = w.Write([]byte(`{}`))
				})
				manager.subsonicRouter = mockRouter
				service.router = manager.subsonicRouter

				req := &subsonicapi.CallRequest{
					Url: "/rest/getAlbum?id=123&u=admin",
				}

				_, err := service.Call(context.Background(), req)

				Expect(err).ToNot(HaveOccurred())
				Expect(capturedRequest).ToNot(BeNil())

				query := capturedRequest.URL.Query()
				Expect(query.Get("c")).To(Equal("test-plugin"))
				Expect(query.Get("f")).To(Equal("json"))
				Expect(query.Get("v")).To(Equal("1.16.1"))
				Expect(query.Get("id")).To(Equal("123"))
				Expect(query.Get("u")).To(Equal("admin"))
			})

			It("should only use path and query from the input URL", func() {
				var capturedRequest *http.Request
				mockRouter = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					capturedRequest = r
					w.WriteHeader(http.StatusOK)
					_, _ = w.Write([]byte(`{}`))
				})
				manager.subsonicRouter = mockRouter
				service.router = manager.subsonicRouter

				req := &subsonicapi.CallRequest{
					Url: "https://external.example.com:8080/rest/ping?u=admin",
				}

				_, err := service.Call(context.Background(), req)

				Expect(err).ToNot(HaveOccurred())
				Expect(capturedRequest).ToNot(BeNil())
				Expect(capturedRequest.URL.Path).To(Equal("/ping"))
				Expect(capturedRequest.URL.Host).To(BeEmpty())
				Expect(capturedRequest.URL.Scheme).To(BeEmpty())
			})

			It("should set internal authentication with username from 'u' parameter", func() {
				var capturedRequest *http.Request
				mockRouter = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					capturedRequest = r
					w.WriteHeader(http.StatusOK)
					_, _ = w.Write([]byte(`{}`))
				})
				manager.subsonicRouter = mockRouter
				service.router = manager.subsonicRouter

				req := &subsonicapi.CallRequest{
					Url: "/rest/ping?u=testuser",
				}

				_, err := service.Call(context.Background(), req)

				Expect(err).ToNot(HaveOccurred())
				Expect(capturedRequest).ToNot(BeNil())

				// Verify that internal authentication is set in the context
				username, ok := request.InternalAuthFrom(capturedRequest.Context())
				Expect(ok).To(BeTrue())
				Expect(username).To(Equal("testuser"))
			})
		})

		Context("when subsonic router is not available", func() {
			BeforeEach(func() {
				manager.subsonicRouter = nil
				service.router = nil
			})

			It("should return an error", func() {
				req := &subsonicapi.CallRequest{
					Url: "/rest/ping?u=admin",
				}

				resp, err := service.Call(context.Background(), req)

				Expect(err).ToNot(HaveOccurred())
				Expect(resp).ToNot(BeNil())
				Expect(resp.Error).To(Equal("SubsonicAPI router not available"))
				Expect(resp.Json).To(BeEmpty())
			})
		})

		Context("when URL is invalid", func() {
			It("should return an error for malformed URLs", func() {
				req := &subsonicapi.CallRequest{
					Url: "://invalid-url",
				}

				resp, err := service.Call(context.Background(), req)

				Expect(err).ToNot(HaveOccurred())
				Expect(resp).ToNot(BeNil())
				Expect(resp.Error).To(ContainSubstring("invalid URL format"))
				Expect(resp.Json).To(BeEmpty())
			})

			It("should return an error when 'u' parameter is missing", func() {
				req := &subsonicapi.CallRequest{
					Url: "/rest/ping?p=password",
				}

				resp, err := service.Call(context.Background(), req)

				Expect(err).ToNot(HaveOccurred())
				Expect(resp).ToNot(BeNil())
				Expect(resp.Error).To(Equal("missing required parameter 'u' (username)"))
				Expect(resp.Json).To(BeEmpty())
			})
		})
	})
})
