package external_test

import (
	"context"
	"errors"
	"net/url"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/conf/configtest"
	"github.com/navidrome/navidrome/core/agents"
	. "github.com/navidrome/navidrome/core/external"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/mock"
)

var _ = Describe("Provider - AlbumImage", func() {
	var ds *tests.MockDataStore
	var provider Provider
	var mockArtistRepo *mockArtistRepo
	var mockAlbumRepo *mockAlbumRepo
	var mockMediaFileRepo *mockMediaFileRepo
	var mockAlbumAgent *mockAlbumInfoAgent
	var ctx context.Context

	BeforeEach(func() {
		ctx = GinkgoT().Context()
		DeferCleanup(configtest.SetupConfig())
		conf.Server.Agents = "mockAlbum" // Configure mock agent

		mockArtistRepo = newMockArtistRepo()
		mockAlbumRepo = newMockAlbumRepo()
		mockMediaFileRepo = newMockMediaFileRepo()

		ds = &tests.MockDataStore{
			MockedArtist:    mockArtistRepo,
			MockedAlbum:     mockAlbumRepo,
			MockedMediaFile: mockMediaFileRepo,
		}

		mockAlbumAgent = newMockAlbumInfoAgent()

		agentsCombined := &mockAgents{albumInfoAgent: mockAlbumAgent}
		provider = NewProvider(ds, agentsCombined)

		// Default mocks
		// Mocks for GetEntityByID sequence (initial failed lookups)
		mockArtistRepo.On("Get", "album-1").Return(nil, model.ErrNotFound).Once()
		mockArtistRepo.On("Get", "mf-1").Return(nil, model.ErrNotFound).Once()
		mockAlbumRepo.On("Get", "mf-1").Return(nil, model.ErrNotFound).Once()

		// Default mock for non-existent entities - Use Maybe() for flexibility
		mockArtistRepo.On("Get", "not-found").Return(nil, model.ErrNotFound).Maybe()
		mockAlbumRepo.On("Get", "not-found").Return(nil, model.ErrNotFound).Maybe()
		mockMediaFileRepo.On("Get", "not-found").Return(nil, model.ErrNotFound).Maybe()
	})

	It("returns the largest image URL when successful", func() {
		// Arrange
		mockArtistRepo.On("Get", "album-1").Return(nil, model.ErrNotFound).Once() // Expect GetEntityByID sequence
		mockAlbumRepo.On("Get", "album-1").Return(&model.Album{ID: "album-1", Name: "Album One", AlbumArtistID: "artist-1"}, nil).Once()
		// Explicitly mock agent call for this test
		mockAlbumAgent.On("GetAlbumImages", ctx, "Album One", "", "").
			Return([]agents.ExternalImage{
				{URL: "http://example.com/large.jpg", Size: 1000},
				{URL: "http://example.com/medium.jpg", Size: 500},
				{URL: "http://example.com/small.jpg", Size: 200},
			}, nil).Once()

		expectedURL, _ := url.Parse("http://example.com/large.jpg")
		imgURL, err := provider.AlbumImage(ctx, "album-1")

		Expect(err).ToNot(HaveOccurred())
		Expect(imgURL).To(Equal(expectedURL))
		mockArtistRepo.AssertCalled(GinkgoT(), "Get", "album-1") // From GetEntityByID
		mockAlbumRepo.AssertCalled(GinkgoT(), "Get", "album-1")
		mockArtistRepo.AssertNotCalled(GinkgoT(), "Get", "artist-1")                       // Artist lookup no longer happens in getAlbum
		mockAlbumAgent.AssertCalled(GinkgoT(), "GetAlbumImages", ctx, "Album One", "", "") // Expect empty artist name
	})

	It("returns ErrNotFound if the album is not found in the DB", func() {
		// Arrange: Explicitly expect the full GetEntityByID sequence for "not-found"
		mockArtistRepo.On("Get", "not-found").Return(nil, model.ErrNotFound).Once()
		mockAlbumRepo.On("Get", "not-found").Return(nil, model.ErrNotFound).Once()
		mockMediaFileRepo.On("Get", "not-found").Return(nil, model.ErrNotFound).Once()

		imgURL, err := provider.AlbumImage(ctx, "not-found")

		Expect(err).To(MatchError("data not found"))
		Expect(imgURL).To(BeNil())
		mockArtistRepo.AssertCalled(GinkgoT(), "Get", "not-found")
		mockAlbumRepo.AssertCalled(GinkgoT(), "Get", "not-found")
		mockMediaFileRepo.AssertCalled(GinkgoT(), "Get", "not-found")
		mockAlbumAgent.AssertNotCalled(GinkgoT(), "GetAlbumImages", mock.Anything, mock.Anything, mock.Anything)
	})

	It("returns the agent error if the agent fails", func() {
		// Arrange
		mockArtistRepo.On("Get", "album-1").Return(nil, model.ErrNotFound).Once() // Expect GetEntityByID sequence
		mockAlbumRepo.On("Get", "album-1").Return(&model.Album{ID: "album-1", Name: "Album One", AlbumArtistID: "artist-1"}, nil).Once()

		agentErr := errors.New("agent failure")
		// Explicitly mock agent call for this test
		mockAlbumAgent.On("GetAlbumImages", ctx, "Album One", "", "").Return(nil, agentErr).Once() // Expect empty artist

		imgURL, err := provider.AlbumImage(ctx, "album-1")

		Expect(err).To(MatchError("agent failure"))
		Expect(imgURL).To(BeNil())
		mockArtistRepo.AssertCalled(GinkgoT(), "Get", "album-1")
		mockAlbumRepo.AssertCalled(GinkgoT(), "Get", "album-1")
		mockArtistRepo.AssertNotCalled(GinkgoT(), "Get", "artist-1")
		mockAlbumAgent.AssertCalled(GinkgoT(), "GetAlbumImages", ctx, "Album One", "", "") // Expect empty artist
	})

	It("returns ErrNotFound if the agent returns ErrNotFound", func() {
		// Arrange
		mockArtistRepo.On("Get", "album-1").Return(nil, model.ErrNotFound).Once() // Expect GetEntityByID sequence
		mockAlbumRepo.On("Get", "album-1").Return(&model.Album{ID: "album-1", Name: "Album One", AlbumArtistID: "artist-1"}, nil).Once()

		// Explicitly mock agent call for this test
		mockAlbumAgent.On("GetAlbumImages", ctx, "Album One", "", "").Return(nil, agents.ErrNotFound).Once() // Expect empty artist

		imgURL, err := provider.AlbumImage(ctx, "album-1")

		Expect(err).To(MatchError("data not found"))
		Expect(imgURL).To(BeNil())
		mockArtistRepo.AssertCalled(GinkgoT(), "Get", "album-1")
		mockAlbumRepo.AssertCalled(GinkgoT(), "Get", "album-1")
		mockAlbumAgent.AssertCalled(GinkgoT(), "GetAlbumImages", ctx, "Album One", "", "") // Expect empty artist
	})

	It("returns ErrNotFound if the agent returns no images", func() {
		// Arrange
		mockArtistRepo.On("Get", "album-1").Return(nil, model.ErrNotFound).Once() // Expect GetEntityByID sequence
		mockAlbumRepo.On("Get", "album-1").Return(&model.Album{ID: "album-1", Name: "Album One", AlbumArtistID: "artist-1"}, nil).Once()

		// Explicitly mock agent call for this test
		mockAlbumAgent.On("GetAlbumImages", ctx, "Album One", "", "").
			Return([]agents.ExternalImage{}, nil).Once() // Expect empty artist

		imgURL, err := provider.AlbumImage(ctx, "album-1")

		Expect(err).To(MatchError("data not found"))
		Expect(imgURL).To(BeNil())
		mockArtistRepo.AssertCalled(GinkgoT(), "Get", "album-1")
		mockAlbumRepo.AssertCalled(GinkgoT(), "Get", "album-1")
		mockAlbumAgent.AssertCalled(GinkgoT(), "GetAlbumImages", ctx, "Album One", "", "") // Expect empty artist
	})

	It("returns context error if context is canceled", func() {
		// Arrange
		cctx, cancelCtx := context.WithCancel(ctx)
		// Mock the necessary DB calls *before* canceling the context
		mockArtistRepo.On("Get", "album-1").Return(nil, model.ErrNotFound).Once()
		mockAlbumRepo.On("Get", "album-1").Return(&model.Album{ID: "album-1", Name: "Album One", AlbumArtistID: "artist-1"}, nil).Once()
		// Expect the agent call even if context is cancelled, returning the context error
		mockAlbumAgent.On("GetAlbumImages", cctx, "Album One", "", "").Return(nil, context.Canceled).Once()
		// Cancel the context *before* calling the function under test
		cancelCtx()

		imgURL, err := provider.AlbumImage(cctx, "album-1")

		Expect(err).To(MatchError("context canceled"))
		Expect(imgURL).To(BeNil())
		mockArtistRepo.AssertCalled(GinkgoT(), "Get", "album-1")
		mockAlbumRepo.AssertCalled(GinkgoT(), "Get", "album-1")
		// Agent should now be called, verify this expectation
		mockAlbumAgent.AssertCalled(GinkgoT(), "GetAlbumImages", cctx, "Album One", "", "")
	})

	It("derives album ID from MediaFile ID", func() {
		// Arrange: Mock full GetEntityByID for "mf-1" and recursive "album-1"
		mockArtistRepo.On("Get", "mf-1").Return(nil, model.ErrNotFound).Once()
		mockAlbumRepo.On("Get", "mf-1").Return(nil, model.ErrNotFound).Once()
		mockMediaFileRepo.On("Get", "mf-1").Return(&model.MediaFile{ID: "mf-1", Title: "Track One", ArtistID: "artist-1", AlbumID: "album-1"}, nil).Once()
		mockArtistRepo.On("Get", "album-1").Return(nil, model.ErrNotFound).Once()
		mockAlbumRepo.On("Get", "album-1").Return(&model.Album{ID: "album-1", Name: "Album One", AlbumArtistID: "artist-1"}, nil).Once()

		// Explicitly mock agent call for this test
		mockAlbumAgent.On("GetAlbumImages", ctx, "Album One", "", "").
			Return([]agents.ExternalImage{
				{URL: "http://example.com/large.jpg", Size: 1000},
				{URL: "http://example.com/medium.jpg", Size: 500},
				{URL: "http://example.com/small.jpg", Size: 200},
			}, nil).Once()

		expectedURL, _ := url.Parse("http://example.com/large.jpg")
		imgURL, err := provider.AlbumImage(ctx, "mf-1")

		Expect(err).ToNot(HaveOccurred())
		Expect(imgURL).To(Equal(expectedURL))
		mockArtistRepo.AssertCalled(GinkgoT(), "Get", "mf-1")
		mockAlbumRepo.AssertCalled(GinkgoT(), "Get", "mf-1")
		mockMediaFileRepo.AssertCalled(GinkgoT(), "Get", "mf-1")
		mockArtistRepo.AssertCalled(GinkgoT(), "Get", "album-1")
		mockAlbumRepo.AssertCalled(GinkgoT(), "Get", "album-1")
		mockArtistRepo.AssertNotCalled(GinkgoT(), "Get", "artist-1")
		mockAlbumAgent.AssertCalled(GinkgoT(), "GetAlbumImages", ctx, "Album One", "", "")
	})

	It("handles different image orders from agent", func() {
		// Arrange
		mockArtistRepo.On("Get", "album-1").Return(nil, model.ErrNotFound).Once() // Expect GetEntityByID sequence
		mockAlbumRepo.On("Get", "album-1").Return(&model.Album{ID: "album-1", Name: "Album One", AlbumArtistID: "artist-1"}, nil).Once()
		// Explicitly mock agent call for this test
		mockAlbumAgent.On("GetAlbumImages", ctx, "Album One", "", "").
			Return([]agents.ExternalImage{
				{URL: "http://example.com/small.jpg", Size: 200},
				{URL: "http://example.com/large.jpg", Size: 1000},
				{URL: "http://example.com/medium.jpg", Size: 500},
			}, nil).Once()

		expectedURL, _ := url.Parse("http://example.com/large.jpg")
		imgURL, err := provider.AlbumImage(ctx, "album-1")

		Expect(err).ToNot(HaveOccurred())
		Expect(imgURL).To(Equal(expectedURL)) // Should still pick the largest
		mockAlbumAgent.AssertCalled(GinkgoT(), "GetAlbumImages", ctx, "Album One", "", "")
	})

	It("handles agent returning only one image", func() {
		// Arrange
		mockArtistRepo.On("Get", "album-1").Return(nil, model.ErrNotFound).Once() // Expect GetEntityByID sequence
		mockAlbumRepo.On("Get", "album-1").Return(&model.Album{ID: "album-1", Name: "Album One", AlbumArtistID: "artist-1"}, nil).Once()
		// Explicitly mock agent call for this test
		mockAlbumAgent.On("GetAlbumImages", ctx, "Album One", "", "").
			Return([]agents.ExternalImage{
				{URL: "http://example.com/single.jpg", Size: 700},
			}, nil).Once()

		expectedURL, _ := url.Parse("http://example.com/single.jpg")
		imgURL, err := provider.AlbumImage(ctx, "album-1")

		Expect(err).ToNot(HaveOccurred())
		Expect(imgURL).To(Equal(expectedURL))
		mockAlbumAgent.AssertCalled(GinkgoT(), "GetAlbumImages", ctx, "Album One", "", "")
	})

	It("returns ErrNotFound if deriving album ID fails", func() {
		// Arrange: Mock full GetEntityByID for "mf-no-album" and recursive "not-found"
		mockArtistRepo.On("Get", "mf-no-album").Return(nil, model.ErrNotFound).Once()
		mockAlbumRepo.On("Get", "mf-no-album").Return(nil, model.ErrNotFound).Once()
		mockMediaFileRepo.On("Get", "mf-no-album").Return(&model.MediaFile{ID: "mf-no-album", Title: "Track No Album", ArtistID: "artist-1", AlbumID: "not-found"}, nil).Once()
		mockArtistRepo.On("Get", "not-found").Return(nil, model.ErrNotFound).Once()
		mockAlbumRepo.On("Get", "not-found").Return(nil, model.ErrNotFound).Once()
		mockMediaFileRepo.On("Get", "not-found").Return(nil, model.ErrNotFound).Once()

		imgURL, err := provider.AlbumImage(ctx, "mf-no-album")

		Expect(err).To(MatchError("data not found"))
		Expect(imgURL).To(BeNil())
		mockArtistRepo.AssertCalled(GinkgoT(), "Get", "mf-no-album")
		mockAlbumRepo.AssertCalled(GinkgoT(), "Get", "mf-no-album")
		mockMediaFileRepo.AssertCalled(GinkgoT(), "Get", "mf-no-album")
		mockArtistRepo.AssertCalled(GinkgoT(), "Get", "not-found")
		mockAlbumRepo.AssertCalled(GinkgoT(), "Get", "not-found")
		mockMediaFileRepo.AssertCalled(GinkgoT(), "Get", "not-found")
		mockAlbumAgent.AssertNotCalled(GinkgoT(), "GetAlbumImages", mock.Anything, mock.Anything, mock.Anything)
	})
})

// mockAlbumInfoAgent implementation
type mockAlbumInfoAgent struct {
	mock.Mock
	agents.AlbumInfoRetriever
	agents.AlbumImageRetriever
}

func newMockAlbumInfoAgent() *mockAlbumInfoAgent {
	m := new(mockAlbumInfoAgent)
	m.On("AgentName").Return("mockAlbum").Maybe()
	return m
}

func (m *mockAlbumInfoAgent) AgentName() string {
	args := m.Called()
	return args.String(0)
}

func (m *mockAlbumInfoAgent) GetAlbumInfo(ctx context.Context, name, artist, mbid string) (*agents.AlbumInfo, error) {
	args := m.Called(ctx, name, artist, mbid)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*agents.AlbumInfo), args.Error(1)
}

func (m *mockAlbumInfoAgent) GetAlbumImages(ctx context.Context, name, artist, mbid string) ([]agents.ExternalImage, error) {
	args := m.Called(ctx, name, artist, mbid)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]agents.ExternalImage), args.Error(1)
}

// Ensure mockAgent implements the interfaces
var _ agents.AlbumInfoRetriever = (*mockAlbumInfoAgent)(nil)
var _ agents.AlbumImageRetriever = (*mockAlbumInfoAgent)(nil)
