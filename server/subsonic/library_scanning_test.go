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

		It("triggers a selective scan with single target parameter", func() {
			// Setup mocks
			mockUserRepo := tests.CreateMockUserRepo()
			_ = mockUserRepo.SetUserLibraries("admin-id", []int{1, 2})
			mockDS := &tests.MockDataStore{MockedUser: mockUserRepo}
			api.ds = mockDS

			// Create admin user
			ctx := request.WithUser(context.Background(), model.User{
				ID:      "admin-id",
				IsAdmin: true,
			})

			// Create request with single target parameter
			r := httptest.NewRequest("GET", "/rest/startScan?target=1:Music/Rock", nil)
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

		It("triggers a selective scan with multiple target parameters", func() {
			// Setup mocks
			mockUserRepo := tests.CreateMockUserRepo()
			_ = mockUserRepo.SetUserLibraries("admin-id", []int{1, 2})
			mockDS := &tests.MockDataStore{MockedUser: mockUserRepo}
			api.ds = mockDS

			// Create admin user
			ctx := request.WithUser(context.Background(), model.User{
				ID:      "admin-id",
				IsAdmin: true,
			})

			// Create request with multiple target parameters
			r := httptest.NewRequest("GET", "/rest/startScan?target=1:Music/Reggae&target=2:Classical/Bach", nil)
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

		It("triggers a selective full scan with target and fullScan parameters", func() {
			// Setup mocks
			mockUserRepo := tests.CreateMockUserRepo()
			_ = mockUserRepo.SetUserLibraries("admin-id", []int{1})
			mockDS := &tests.MockDataStore{MockedUser: mockUserRepo}
			api.ds = mockDS

			// Create admin user
			ctx := request.WithUser(context.Background(), model.User{
				ID:      "admin-id",
				IsAdmin: true,
			})

			// Create request with target and fullScan parameters
			r := httptest.NewRequest("GET", "/rest/startScan?target=1:Music/Jazz&fullScan=true", nil)
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

		It("returns error for invalid target format", func() {
			// Create admin user
			ctx := request.WithUser(context.Background(), model.User{
				ID:      "admin-id",
				IsAdmin: true,
			})

			// Create request with invalid target format (missing colon)
			r := httptest.NewRequest("GET", "/rest/startScan?target=1MusicRock", nil)
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

		It("returns error for invalid library ID in target", func() {
			// Create admin user
			ctx := request.WithUser(context.Background(), model.User{
				ID:      "admin-id",
				IsAdmin: true,
			})

			// Create request with invalid library ID
			r := httptest.NewRequest("GET", "/rest/startScan?target=0:Music/Rock", nil)
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

		It("returns error when library does not exist", func() {
			// Setup mocks - user has access to library 1 and 2 only
			mockUserRepo := tests.CreateMockUserRepo()
			_ = mockUserRepo.SetUserLibraries("admin-id", []int{1, 2})
			mockDS := &tests.MockDataStore{MockedUser: mockUserRepo}
			api.ds = mockDS

			// Create admin user
			ctx := request.WithUser(context.Background(), model.User{
				ID:      "admin-id",
				IsAdmin: true,
			})

			// Create request with library ID that doesn't exist
			r := httptest.NewRequest("GET", "/rest/startScan?target=999:Music/Rock", nil)
			r = r.WithContext(ctx)

			// Call endpoint
			response, err := api.StartScan(r)

			// Should return ErrorDataNotFound
			Expect(err).To(HaveOccurred())
			Expect(response).To(BeNil())
			var subErr subError
			ok := errors.As(err, &subErr)
			Expect(ok).To(BeTrue())
			Expect(subErr.code).To(Equal(responses.ErrorDataNotFound))
		})

		It("calls ScanAll when single library with empty path and only one library exists", func() {
			// Setup mocks - single library in DB
			mockUserRepo := tests.CreateMockUserRepo()
			_ = mockUserRepo.SetUserLibraries("admin-id", []int{1})
			mockLibraryRepo := &tests.MockLibraryRepo{}
			mockLibraryRepo.SetData(model.Libraries{
				{ID: 1, Name: "Music Library", Path: "/music"},
			})
			mockDS := &tests.MockDataStore{
				MockedUser:    mockUserRepo,
				MockedLibrary: mockLibraryRepo,
			}
			api.ds = mockDS

			// Create admin user
			ctx := request.WithUser(context.Background(), model.User{
				ID:      "admin-id",
				IsAdmin: true,
			})

			// Create request with single library and empty path
			r := httptest.NewRequest("GET", "/rest/startScan?target=1:", nil)
			r = r.WithContext(ctx)

			// Call endpoint
			response, err := api.StartScan(r)

			// Should succeed
			Expect(err).ToNot(HaveOccurred())
			Expect(response).ToNot(BeNil())

			// Verify ScanAll was called instead of ScanFolders
			Eventually(func() int {
				return ms.GetScanAllCallCount()
			}).Should(BeNumerically(">", 0))
			Expect(ms.GetScanFoldersCallCount()).To(Equal(0))
		})

		It("calls ScanFolders when single library with empty path but multiple libraries exist", func() {
			// Setup mocks - multiple libraries in DB
			mockUserRepo := tests.CreateMockUserRepo()
			_ = mockUserRepo.SetUserLibraries("admin-id", []int{1, 2})
			mockLibraryRepo := &tests.MockLibraryRepo{}
			mockLibraryRepo.SetData(model.Libraries{
				{ID: 1, Name: "Music Library", Path: "/music"},
				{ID: 2, Name: "Audiobooks", Path: "/audiobooks"},
			})
			mockDS := &tests.MockDataStore{
				MockedUser:    mockUserRepo,
				MockedLibrary: mockLibraryRepo,
			}
			api.ds = mockDS

			// Create admin user
			ctx := request.WithUser(context.Background(), model.User{
				ID:      "admin-id",
				IsAdmin: true,
			})

			// Create request with single library and empty path
			r := httptest.NewRequest("GET", "/rest/startScan?target=1:", nil)
			r = r.WithContext(ctx)

			// Call endpoint
			response, err := api.StartScan(r)

			// Should succeed
			Expect(err).ToNot(HaveOccurred())
			Expect(response).ToNot(BeNil())

			// Verify ScanFolders was called (not ScanAll)
			Eventually(func() int {
				return ms.GetScanFoldersCallCount()
			}).Should(BeNumerically(">", 0))
			calls := ms.GetScanFoldersCalls()
			Expect(calls).To(HaveLen(1))
			targets := calls[0].Targets
			Expect(targets).To(HaveLen(1))
			Expect(targets[0].LibraryID).To(Equal(1))
			Expect(targets[0].FolderPath).To(Equal(""))
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
