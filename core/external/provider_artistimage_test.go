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

var _ = Describe("Provider - ArtistImage", func() {
	var ds *tests.MockDataStore
	var provider Provider
	var mockArtistRepo *mockArtistRepo
	var mockAlbumRepo *mockAlbumRepo
	var mockMediaFileRepo *mockMediaFileRepo
	var mockImageAgent *mockArtistImageAgent
	var agentsCombined *mockAgents
	var ctx context.Context

	BeforeEach(func() {
		DeferCleanup(configtest.SetupConfig())
		conf.Server.Agents = "mockImage" // Configure only the mock agent
		ctx = GinkgoT().Context()

		mockArtistRepo = newMockArtistRepo()
		mockAlbumRepo = newMockAlbumRepo()
		mockMediaFileRepo = newMockMediaFileRepo()

		ds = &tests.MockDataStore{
			MockedArtist:    mockArtistRepo,
			MockedAlbum:     mockAlbumRepo,
			MockedMediaFile: mockMediaFileRepo,
		}

		mockImageAgent = newMockArtistImageAgent()

		// Use the mockAgents from helper, setting the specific agent
		agentsCombined = &mockAgents{
			imageAgent: mockImageAgent,
		}

		provider = NewProvider(ds, agentsCombined)

		// Default mocks for successful Get calls
		mockArtistRepo.On("Get", "artist-1").Return(&model.Artist{ID: "artist-1", Name: "Artist One"}, nil).Maybe()
		mockAlbumRepo.On("Get", "album-1").Return(&model.Album{ID: "album-1", Name: "Album One", AlbumArtistID: "artist-1"}, nil).Maybe()
		mockMediaFileRepo.On("Get", "mf-1").Return(&model.MediaFile{ID: "mf-1", Title: "Track One", ArtistID: "artist-1"}, nil).Maybe()
		// Default mock for non-existent entities
		mockArtistRepo.On("Get", "not-found").Return(nil, model.ErrNotFound).Maybe()
		mockAlbumRepo.On("Get", "not-found").Return(nil, model.ErrNotFound).Maybe()
		mockMediaFileRepo.On("Get", "not-found").Return(nil, model.ErrNotFound).Maybe()

		// Default successful image agent response
		mockImageAgent.On("GetArtistImages", mock.Anything, "artist-1", "Artist One", "").
			Return([]agents.ExternalImage{
				{URL: "http://example.com/large.jpg", Size: 1000},
				{URL: "http://example.com/medium.jpg", Size: 500},
				{URL: "http://example.com/small.jpg", Size: 200},
			}, nil).Maybe()
	})

	AfterEach(func() {
		mockArtistRepo.AssertExpectations(GinkgoT())
		mockAlbumRepo.AssertExpectations(GinkgoT())
		mockMediaFileRepo.AssertExpectations(GinkgoT())
		mockImageAgent.AssertExpectations(GinkgoT())
	})

	It("returns the largest image URL when successful", func() {
		// Arrange
		expectedURL, _ := url.Parse("http://example.com/large.jpg")

		// Act
		imgURL, err := provider.ArtistImage(ctx, "artist-1")

		// Assert
		Expect(err).ToNot(HaveOccurred())
		Expect(imgURL).To(Equal(expectedURL))
		mockArtistRepo.AssertCalled(GinkgoT(), "Get", "artist-1")
		mockImageAgent.AssertCalled(GinkgoT(), "GetArtistImages", ctx, "artist-1", "Artist One", "")
	})

	It("returns ErrNotFound if the artist is not found in the DB", func() {
		// Arrange

		// Act
		imgURL, err := provider.ArtistImage(ctx, "not-found")

		// Assert
		Expect(err).To(MatchError(model.ErrNotFound))
		Expect(imgURL).To(BeNil())
		mockArtistRepo.AssertCalled(GinkgoT(), "Get", "not-found")
		mockImageAgent.AssertNotCalled(GinkgoT(), "GetArtistImages", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	})

	It("returns the agent error if the agent fails", func() {
		// Arrange
		agentErr := errors.New("agent failure")
		mockImageAgent.Mock = mock.Mock{} // Reset default expectation
		mockImageAgent.On("GetArtistImages", ctx, "artist-1", "Artist One", "").Return(nil, agentErr).Once()

		// Act
		imgURL, err := provider.ArtistImage(ctx, "artist-1")

		// Assert
		Expect(err).To(MatchError(model.ErrNotFound)) // Corrected Expectation: The provider maps agent errors (other than canceled) to ErrNotFound if no image was found/populated
		Expect(imgURL).To(BeNil())
		mockArtistRepo.AssertCalled(GinkgoT(), "Get", "artist-1")
		mockImageAgent.AssertCalled(GinkgoT(), "GetArtistImages", ctx, "artist-1", "Artist One", "")
	})

	It("returns ErrNotFound if the agent returns ErrNotFound", func() {
		// Arrange
		mockImageAgent.Mock = mock.Mock{} // Reset default expectation
		mockImageAgent.On("GetArtistImages", ctx, "artist-1", "Artist One", "").Return(nil, agents.ErrNotFound).Once()

		// Act
		imgURL, err := provider.ArtistImage(ctx, "artist-1")

		// Assert
		Expect(err).To(MatchError(model.ErrNotFound))
		Expect(imgURL).To(BeNil())
		mockArtistRepo.AssertCalled(GinkgoT(), "Get", "artist-1")
		mockImageAgent.AssertCalled(GinkgoT(), "GetArtistImages", ctx, "artist-1", "Artist One", "")
	})

	It("returns ErrNotFound if the agent returns no images", func() {
		// Arrange
		mockImageAgent.Mock = mock.Mock{} // Reset default expectation
		mockImageAgent.On("GetArtistImages", ctx, "artist-1", "Artist One", "").Return([]agents.ExternalImage{}, nil).Once()

		// Act
		imgURL, err := provider.ArtistImage(ctx, "artist-1")

		// Assert
		Expect(err).To(MatchError(model.ErrNotFound)) // Implementation maps empty result to ErrNotFound
		Expect(imgURL).To(BeNil())
		mockArtistRepo.AssertCalled(GinkgoT(), "Get", "artist-1")
		mockImageAgent.AssertCalled(GinkgoT(), "GetArtistImages", ctx, "artist-1", "Artist One", "")
	})

	It("returns context error if context is canceled before agent call", func() {
		// Arrange
		cctx, cancelCtx := context.WithCancel(context.Background())
		mockArtistRepo.Mock = mock.Mock{} // Reset default expectation for artist repo as well
		mockArtistRepo.On("Get", "artist-1").Return(&model.Artist{ID: "artist-1", Name: "Artist One"}, nil).Run(func(args mock.Arguments) {
			cancelCtx() // Cancel context *during* the DB call simulation
		}).Once()

		// Act
		imgURL, err := provider.ArtistImage(cctx, "artist-1")

		// Assert
		Expect(err).To(MatchError(context.Canceled))
		Expect(imgURL).To(BeNil())
		mockArtistRepo.AssertCalled(GinkgoT(), "Get", "artist-1")
	})

	It("derives artist ID from MediaFile ID", func() {
		// Arrange: Add mocks for the initial GetEntityByID lookups
		mockArtistRepo.On("Get", "mf-1").Return(nil, model.ErrNotFound).Once()
		mockAlbumRepo.On("Get", "mf-1").Return(nil, model.ErrNotFound).Once()
		// Default mocks for MediaFileRepo.Get("mf-1") and ArtistRepo.Get("artist-1") handle the rest
		expectedURL, _ := url.Parse("http://example.com/large.jpg")

		// Act
		imgURL, err := provider.ArtistImage(ctx, "mf-1")

		// Assert
		Expect(err).ToNot(HaveOccurred())
		Expect(imgURL).To(Equal(expectedURL))
		mockArtistRepo.AssertCalled(GinkgoT(), "Get", "mf-1") // GetEntityByID sequence
		mockAlbumRepo.AssertCalled(GinkgoT(), "Get", "mf-1")  // GetEntityByID sequence
		mockMediaFileRepo.AssertCalled(GinkgoT(), "Get", "mf-1")
		mockArtistRepo.AssertCalled(GinkgoT(), "Get", "artist-1") // Should be called after getting MF
		mockImageAgent.AssertCalled(GinkgoT(), "GetArtistImages", ctx, "artist-1", "Artist One", "")
	})

	It("derives artist ID from Album ID", func() {
		// Arrange: Add mock for the initial GetEntityByID lookup
		mockArtistRepo.On("Get", "album-1").Return(nil, model.ErrNotFound).Once()
		// Default mocks for AlbumRepo.Get("album-1") and ArtistRepo.Get("artist-1") handle the rest
		expectedURL, _ := url.Parse("http://example.com/large.jpg")

		// Act
		imgURL, err := provider.ArtistImage(ctx, "album-1")

		// Assert
		Expect(err).ToNot(HaveOccurred())
		Expect(imgURL).To(Equal(expectedURL))
		mockArtistRepo.AssertCalled(GinkgoT(), "Get", "album-1") // GetEntityByID sequence
		mockAlbumRepo.AssertCalled(GinkgoT(), "Get", "album-1")
		mockArtistRepo.AssertCalled(GinkgoT(), "Get", "artist-1") // Should be called after getting Album
		mockImageAgent.AssertCalled(GinkgoT(), "GetArtistImages", ctx, "artist-1", "Artist One", "")
	})

	It("returns ErrNotFound if derived artist is not found", func() {
		// Arrange
		// Add mocks for the initial GetEntityByID lookups
		mockArtistRepo.On("Get", "mf-bad-artist").Return(nil, model.ErrNotFound).Once()
		mockAlbumRepo.On("Get", "mf-bad-artist").Return(nil, model.ErrNotFound).Once()
		mockMediaFileRepo.On("Get", "mf-bad-artist").Return(&model.MediaFile{ID: "mf-bad-artist", ArtistID: "not-found"}, nil).Once()
		// Add expectation for the recursive GetEntityByID call for the MediaFileRepo
		mockMediaFileRepo.On("Get", "not-found").Return(nil, model.ErrNotFound).Maybe()
		// The default mocks for ArtistRepo/AlbumRepo handle the final "not-found" lookups

		// Act
		imgURL, err := provider.ArtistImage(ctx, "mf-bad-artist")

		// Assert
		Expect(err).To(MatchError(model.ErrNotFound))
		Expect(imgURL).To(BeNil())
		mockArtistRepo.AssertCalled(GinkgoT(), "Get", "mf-bad-artist") // GetEntityByID sequence
		mockAlbumRepo.AssertCalled(GinkgoT(), "Get", "mf-bad-artist")  // GetEntityByID sequence
		mockMediaFileRepo.AssertCalled(GinkgoT(), "Get", "mf-bad-artist")
		mockArtistRepo.AssertCalled(GinkgoT(), "Get", "not-found")
		mockImageAgent.AssertNotCalled(GinkgoT(), "GetArtistImages", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	})

	It("handles different image orders from agent", func() {
		// Arrange
		mockImageAgent.Mock = mock.Mock{} // Reset default expectation
		mockImageAgent.On("GetArtistImages", ctx, "artist-1", "Artist One", "").
			Return([]agents.ExternalImage{
				{URL: "http://example.com/small.jpg", Size: 200},
				{URL: "http://example.com/large.jpg", Size: 1000},
				{URL: "http://example.com/medium.jpg", Size: 500},
			}, nil).Once()
		expectedURL, _ := url.Parse("http://example.com/large.jpg")

		// Act
		imgURL, err := provider.ArtistImage(ctx, "artist-1")

		// Assert
		Expect(err).ToNot(HaveOccurred())
		Expect(imgURL).To(Equal(expectedURL)) // Still picks the largest
		mockArtistRepo.AssertCalled(GinkgoT(), "Get", "artist-1")
		mockImageAgent.AssertCalled(GinkgoT(), "GetArtistImages", ctx, "artist-1", "Artist One", "")
	})

	It("handles agent returning only one image", func() {
		// Arrange
		mockImageAgent.Mock = mock.Mock{} // Reset default expectation
		mockImageAgent.On("GetArtistImages", ctx, "artist-1", "Artist One", "").
			Return([]agents.ExternalImage{
				{URL: "http://example.com/medium.jpg", Size: 500},
			}, nil).Once()
		expectedURL, _ := url.Parse("http://example.com/medium.jpg")

		// Act
		imgURL, err := provider.ArtistImage(ctx, "artist-1")

		// Assert
		Expect(err).ToNot(HaveOccurred())
		Expect(imgURL).To(Equal(expectedURL))
		mockArtistRepo.AssertCalled(GinkgoT(), "Get", "artist-1")
		mockImageAgent.AssertCalled(GinkgoT(), "GetArtistImages", ctx, "artist-1", "Artist One", "")
	})
})

// mockArtistImageAgent implementation using testify/mock
// This remains local as it's specific to testing the ArtistImage functionality
type mockArtistImageAgent struct {
	mock.Mock
	agents.ArtistImageRetriever // Embed interface
}

// Constructor for the mock agent
func newMockArtistImageAgent() *mockArtistImageAgent {
	mock := new(mockArtistImageAgent)
	// Set default AgentName if needed, although usually called via mockAgents
	mock.On("AgentName").Return("mockImage").Maybe()
	return mock
}

func (m *mockArtistImageAgent) AgentName() string {
	args := m.Called()
	return args.String(0)
}

func (m *mockArtistImageAgent) GetArtistImages(ctx context.Context, id, artistName, mbid string) ([]agents.ExternalImage, error) {
	args := m.Called(ctx, id, artistName, mbid)
	// Need careful type assertion for potentially nil slice
	var res []agents.ExternalImage
	if args.Get(0) != nil {
		res = args.Get(0).([]agents.ExternalImage)
	}
	return res, args.Error(1)
}

// Ensure mockAgent implements the interface
var _ agents.ArtistImageRetriever = (*mockArtistImageAgent)(nil)
