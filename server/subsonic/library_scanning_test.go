package subsonic

import (
	"context"
	"errors"
	"net/http/httptest"

	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
	"github.com/navidrome/navidrome/server/subsonic/responses"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("LibraryScanning", func() {
	var api *Router
	var ms *tests.MockScanner

	BeforeEach(func() {
		ms = tests.NewMockScanner()
		api = &Router{scanner: ms}
	})

	Describe("StartScan", func() {
		It("requires admin authentication", func() {
			// Create non-admin user
			ctx := request.WithUser(context.Background(), model.User{
				ID:      "user-id",
				IsAdmin: false,
			})

			// Create request
			r := httptest.NewRequest("GET", "/rest/startScan", nil)
			r = r.WithContext(ctx)

			// Call endpoint
			response, err := api.StartScan(r)

			// Should return authorization error
			Expect(err).To(HaveOccurred())
			Expect(response).To(BeNil())
			var subErr subError
			ok := errors.As(err, &subErr)
			Expect(ok).To(BeTrue())
			Expect(subErr.code).To(Equal(responses.ErrorAuthorizationFail))
		})

		It("triggers a full scan with no parameters", func() {
			// Create admin user
			ctx := request.WithUser(context.Background(), model.User{
				ID:      "admin-id",
				IsAdmin: true,
			})

			// Create request with no parameters
			r := httptest.NewRequest("GET", "/rest/startScan", nil)
			r = r.WithContext(ctx)

			// Call endpoint
			response, err := api.StartScan(r)

			// Should succeed
			Expect(err).ToNot(HaveOccurred())
			Expect(response).ToNot(BeNil())

			// Verify ScanAll was called (eventually, since it's in a goroutine)
			Eventually(func() int {
				return ms.GetScanAllCallCount()
			}).Should(BeNumerically(">", 0))
			calls := ms.GetScanAllCalls()
			Expect(calls).To(HaveLen(1))
			Expect(calls[0].FullScan).To(BeFalse())
		})

		It("triggers a full scan with fullScan=true", func() {
			// Create admin user
			ctx := request.WithUser(context.Background(), model.User{
				ID:      "admin-id",
				IsAdmin: true,
			})

			// Create request with fullScan parameter
			r := httptest.NewRequest("GET", "/rest/startScan?fullScan=true", nil)
			r = r.WithContext(ctx)

			// Call endpoint
			response, err := api.StartScan(r)

			// Should succeed
			Expect(err).ToNot(HaveOccurred())
			Expect(response).ToNot(BeNil())

			// Verify ScanAll was called with fullScan=true
			Eventually(func() int {
				return ms.GetScanAllCallCount()
			}).Should(BeNumerically(">", 0))
			calls := ms.GetScanAllCalls()
			Expect(calls).To(HaveLen(1))
			Expect(calls[0].FullScan).To(BeTrue())
		})

		It("triggers a selective scan with single path parameter", func() {
			// Create admin user
			ctx := request.WithUser(context.Background(), model.User{
				ID:      "admin-id",
				IsAdmin: true,
			})

			// Create request with single path parameter
			r := httptest.NewRequest("GET", "/rest/startScan?path=1:Music/Rock", nil)
			r = r.WithContext(ctx)

			// Call endpoint
			response, err := api.StartScan(r)

			// Should succeed
			Expect(err).ToNot(HaveOccurred())
			Expect(response).ToNot(BeNil())

			// Verify ScanFolders was called with correct targets
			Eventually(func() int {
				return ms.GetScanFoldersCallCount()
			}).Should(BeNumerically(">", 0))
			calls := ms.GetScanFoldersCalls()
			Expect(calls).To(HaveLen(1))
			targets := calls[0].Targets
			Expect(targets).To(HaveLen(1))
			Expect(targets[0].LibraryID).To(Equal(1))
			Expect(targets[0].FolderPath).To(Equal("Music/Rock"))
		})

		It("triggers a selective scan with multiple path parameters", func() {
			// Create admin user
			ctx := request.WithUser(context.Background(), model.User{
				ID:      "admin-id",
				IsAdmin: true,
			})

			// Create request with multiple path parameters
			r := httptest.NewRequest("GET", "/rest/startScan?path=1:Music/Reggae&path=2:Classical/Bach", nil)
			r = r.WithContext(ctx)

			// Call endpoint
			response, err := api.StartScan(r)

			// Should succeed
			Expect(err).ToNot(HaveOccurred())
			Expect(response).ToNot(BeNil())

			// Verify ScanFolders was called with correct targets
			Eventually(func() int {
				return ms.GetScanFoldersCallCount()
			}).Should(BeNumerically(">", 0))
			calls := ms.GetScanFoldersCalls()
			Expect(calls).To(HaveLen(1))
			targets := calls[0].Targets
			Expect(targets).To(HaveLen(2))
			Expect(targets[0].LibraryID).To(Equal(1))
			Expect(targets[0].FolderPath).To(Equal("Music/Reggae"))
			Expect(targets[1].LibraryID).To(Equal(2))
			Expect(targets[1].FolderPath).To(Equal("Classical/Bach"))
		})

		It("triggers a selective full scan with path and fullScan parameters", func() {
			// Create admin user
			ctx := request.WithUser(context.Background(), model.User{
				ID:      "admin-id",
				IsAdmin: true,
			})

			// Create request with path and fullScan parameters
			r := httptest.NewRequest("GET", "/rest/startScan?path=1:Music/Jazz&fullScan=true", nil)
			r = r.WithContext(ctx)

			// Call endpoint
			response, err := api.StartScan(r)

			// Should succeed
			Expect(err).ToNot(HaveOccurred())
			Expect(response).ToNot(BeNil())

			// Verify ScanFolders was called with fullScan=true
			Eventually(func() int {
				return ms.GetScanFoldersCallCount()
			}).Should(BeNumerically(">", 0))
			calls := ms.GetScanFoldersCalls()
			Expect(calls).To(HaveLen(1))
			Expect(calls[0].FullScan).To(BeTrue())
			targets := calls[0].Targets
			Expect(targets).To(HaveLen(1))
		})

		It("returns error for invalid path format", func() {
			// Create admin user
			ctx := request.WithUser(context.Background(), model.User{
				ID:      "admin-id",
				IsAdmin: true,
			})

			// Create request with invalid path format (missing colon)
			r := httptest.NewRequest("GET", "/rest/startScan?path=1MusicRock", nil)
			r = r.WithContext(ctx)

			// Call endpoint
			response, err := api.StartScan(r)

			// Should return error
			Expect(err).To(HaveOccurred())
			Expect(response).To(BeNil())
			var subErr subError
			ok := errors.As(err, &subErr)
			Expect(ok).To(BeTrue())
			Expect(subErr.code).To(Equal(responses.ErrorGeneric))
		})

		It("returns error for invalid library ID", func() {
			// Create admin user
			ctx := request.WithUser(context.Background(), model.User{
				ID:      "admin-id",
				IsAdmin: true,
			})

			// Create request with invalid library ID
			r := httptest.NewRequest("GET", "/rest/startScan?path=0:Music/Rock", nil)
			r = r.WithContext(ctx)

			// Call endpoint
			response, err := api.StartScan(r)

			// Should return error
			Expect(err).To(HaveOccurred())
			Expect(response).To(BeNil())
			var subErr subError
			ok := errors.As(err, &subErr)
			Expect(ok).To(BeTrue())
			Expect(subErr.code).To(Equal(responses.ErrorGeneric))
		})

		It("handles URL-encoded paths", func() {
			// Create admin user
			ctx := request.WithUser(context.Background(), model.User{
				ID:      "admin-id",
				IsAdmin: true,
			})

			// Create request with URL-encoded path
			r := httptest.NewRequest("GET", "/rest/startScan?path=1:The%20Beatles", nil)
			r = r.WithContext(ctx)

			// Call endpoint
			response, err := api.StartScan(r)

			// Should succeed
			Expect(err).ToNot(HaveOccurred())
			Expect(response).ToNot(BeNil())

			// Verify path was decoded correctly
			Eventually(func() int {
				return ms.GetScanFoldersCallCount()
			}).Should(BeNumerically(">", 0))
			calls := ms.GetScanFoldersCalls()
			targets := calls[0].Targets
			Expect(targets[0].FolderPath).To(Equal("The Beatles"))
		})
	})

	Describe("GetScanStatus", func() {
		It("returns scan status", func() {
			// Setup mock scanner status
			ms.SetStatusResponse(&model.ScannerStatus{
				Scanning:    false,
				Count:       100,
				FolderCount: 10,
			})

			// Create request
			ctx := context.Background()
			r := httptest.NewRequest("GET", "/rest/getScanStatus", nil)
			r = r.WithContext(ctx)

			// Call endpoint
			response, err := api.GetScanStatus(r)

			// Should succeed
			Expect(err).ToNot(HaveOccurred())
			Expect(response).ToNot(BeNil())
			Expect(response.ScanStatus).ToNot(BeNil())
			Expect(response.ScanStatus.Scanning).To(BeFalse())
			Expect(response.ScanStatus.Count).To(Equal(int64(100)))
			Expect(response.ScanStatus.FolderCount).To(Equal(int64(10)))
		})
	})
})
