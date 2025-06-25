package plugins

import (
	"context"
	"net/http"

	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
	"github.com/navidrome/navidrome/plugins/host/subsonicapi"
	"github.com/navidrome/navidrome/plugins/schema"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("SubsonicAPI Host Service", func() {
	var (
		service    *subsonicAPIServiceImpl
		mockRouter http.Handler
		userRepo   *tests.MockedUserRepo
	)

	BeforeEach(func() {
		// Setup mock datastore with users
		userRepo = tests.CreateMockUserRepo()
		_ = userRepo.Put(&model.User{UserName: "admin", IsAdmin: true})
		_ = userRepo.Put(&model.User{UserName: "user", IsAdmin: false})
		ds := &tests.MockDataStore{MockedUser: userRepo}

		// Create a mock router
		mockRouter = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"subsonic-response":{"status":"ok","version":"1.16.1"}}`))
		})

		// Create service implementation
		service = &subsonicAPIServiceImpl{
			pluginID: "test-plugin",
			router:   mockRouter,
			ds:       ds,
		}
	})

	// Helper function to create a mock router that captures the request
	setupRequestCapture := func() **http.Request {
		var capturedRequest *http.Request
		mockRouter = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			capturedRequest = r
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{}`))
		})
		service.router = mockRouter
		return &capturedRequest
	}

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
				capturedRequestPtr := setupRequestCapture()

				req := &subsonicapi.CallRequest{
					Url: "/rest/getAlbum.view?id=123&u=admin",
				}

				_, err := service.Call(context.Background(), req)

				Expect(err).ToNot(HaveOccurred())
				Expect(*capturedRequestPtr).ToNot(BeNil())

				query := (*capturedRequestPtr).URL.Query()
				Expect(query.Get("c")).To(Equal("test-plugin"))
				Expect(query.Get("f")).To(Equal("json"))
				Expect(query.Get("v")).To(Equal("1.16.1"))
				Expect(query.Get("id")).To(Equal("123"))
				Expect(query.Get("u")).To(Equal("admin"))
			})

			It("should only use path and query from the input URL", func() {
				capturedRequestPtr := setupRequestCapture()

				req := &subsonicapi.CallRequest{
					Url: "https://external.example.com:8080/rest/ping?u=admin",
				}

				_, err := service.Call(context.Background(), req)

				Expect(err).ToNot(HaveOccurred())
				Expect(*capturedRequestPtr).ToNot(BeNil())
				Expect((*capturedRequestPtr).URL.Path).To(Equal("/ping"))
				Expect((*capturedRequestPtr).URL.Host).To(BeEmpty())
				Expect((*capturedRequestPtr).URL.Scheme).To(BeEmpty())
			})

			It("ignores the path prefix in the URL", func() {
				capturedRequestPtr := setupRequestCapture()

				req := &subsonicapi.CallRequest{
					Url: "/basepath/rest/ping?u=admin",
				}

				_, err := service.Call(context.Background(), req)

				Expect(err).ToNot(HaveOccurred())
				Expect(*capturedRequestPtr).ToNot(BeNil())
				Expect((*capturedRequestPtr).URL.Path).To(Equal("/ping"))
			})

			It("should set internal authentication with username from 'u' parameter", func() {
				capturedRequestPtr := setupRequestCapture()

				req := &subsonicapi.CallRequest{
					Url: "/rest/ping?u=testuser",
				}

				_, err := service.Call(context.Background(), req)

				Expect(err).ToNot(HaveOccurred())
				Expect(*capturedRequestPtr).ToNot(BeNil())

				// Verify that internal authentication is set in the context
				username, ok := request.InternalAuthFrom((*capturedRequestPtr).Context())
				Expect(ok).To(BeTrue())
				Expect(username).To(Equal("testuser"))
			})
		})

		Context("when subsonic router is not available", func() {
			BeforeEach(func() {
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

		Context("permission checks", func() {
			It("rejects disallowed username", func() {
				service.permissions = parseSubsonicAPIPermissions(&schema.PluginManifestPermissionsSubsonicapi{
					Reason:           "test",
					AllowedUsernames: []string{"user"},
				})

				resp, err := service.Call(context.Background(), &subsonicapi.CallRequest{Url: "/rest/ping?u=admin"})
				Expect(err).ToNot(HaveOccurred())
				Expect(resp.Error).To(ContainSubstring("not allowed"))
			})

			It("rejects admin when allowAdmins is false", func() {
				service.permissions = parseSubsonicAPIPermissions(&schema.PluginManifestPermissionsSubsonicapi{Reason: "test"})

				resp, err := service.Call(context.Background(), &subsonicapi.CallRequest{Url: "/rest/ping?u=admin"})
				Expect(err).ToNot(HaveOccurred())
				Expect(resp.Error).To(ContainSubstring("not allowed"))
			})

			It("allows admin when allowAdmins is true", func() {
				service.permissions = parseSubsonicAPIPermissions(&schema.PluginManifestPermissionsSubsonicapi{Reason: "test", AllowAdmins: true})

				resp, err := service.Call(context.Background(), &subsonicapi.CallRequest{Url: "/rest/ping?u=admin"})
				Expect(err).ToNot(HaveOccurred())
				Expect(resp.Error).To(BeEmpty())
			})
		})
	})
})
