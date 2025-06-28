package external_test

import (
	"context"
	"errors"

	"github.com/navidrome/navidrome/core/agents"
	. "github.com/navidrome/navidrome/core/external"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/mock"
)

var _ = Describe("Provider - ArtistRadio", func() {
	var ds model.DataStore
	var provider Provider
	var mockAgent *mockSimilarArtistAgent
	var mockTopAgent agents.ArtistTopSongsRetriever
	var mockSimilarAgent agents.ArtistSimilarRetriever
	var agentsCombined Agents
	var artistRepo *mockArtistRepo
	var mediaFileRepo *mockMediaFileRepo
	var ctx context.Context

	BeforeEach(func() {
		ctx = GinkgoT().Context()

		artistRepo = newMockArtistRepo()
		mediaFileRepo = newMockMediaFileRepo()

		ds = &tests.MockDataStore{
			MockedArtist:    artistRepo,
			MockedMediaFile: mediaFileRepo,
		}

		mockAgent = &mockSimilarArtistAgent{}
		mockTopAgent = mockAgent
		mockSimilarAgent = mockAgent

		agentsCombined = &mockAgents{
			topSongsAgent: mockTopAgent,
			similarAgent:  mockSimilarAgent,
		}

		provider = NewProvider(ds, agentsCombined)
	})

	It("returns similar songs from main artist and similar artists", func() {
		artist1 := model.Artist{ID: "artist-1", Name: "Artist One"}
		similarArtist := model.Artist{ID: "artist-3", Name: "Similar Artist"}
		song1 := model.MediaFile{ID: "song-1", Title: "Song One", ArtistID: "artist-1", MbzRecordingID: "mbid-1"}
		song2 := model.MediaFile{ID: "song-2", Title: "Song Two", ArtistID: "artist-1", MbzRecordingID: "mbid-2"}
		song3 := model.MediaFile{ID: "song-3", Title: "Song Three", ArtistID: "artist-3", MbzRecordingID: "mbid-3"}

		artistRepo.On("Get", "artist-1").Return(&artist1, nil).Maybe()
		artistRepo.On("Get", "artist-3").Return(&similarArtist, nil).Maybe()

		artistRepo.On("GetAll", mock.MatchedBy(func(opt model.QueryOptions) bool {
			return opt.Max == 1 && opt.Filters != nil
		})).Return(model.Artists{artist1}, nil).Once()

		similarAgentsResp := []agents.Artist{
			{Name: "Similar Artist", MBID: "similar-mbid"},
		}
		mockAgent.On("GetSimilarArtists", mock.Anything, "artist-1", "Artist One", "", 15).
			Return(similarAgentsResp, nil).Once()

		artistRepo.On("GetAll", mock.MatchedBy(func(opt model.QueryOptions) bool {
			return opt.Max == 0 && opt.Filters != nil
		})).Return(model.Artists{similarArtist}, nil).Once()

		mockAgent.On("GetArtistTopSongs", mock.Anything, "artist-1", "Artist One", "", mock.Anything).
			Return([]agents.Song{
				{Name: "Song One", MBID: "mbid-1"},
				{Name: "Song Two", MBID: "mbid-2"},
			}, nil).Once()

		mockAgent.On("GetArtistTopSongs", mock.Anything, "artist-3", "Similar Artist", "", mock.Anything).
			Return([]agents.Song{
				{Name: "Song Three", MBID: "mbid-3"},
			}, nil).Once()

		mediaFileRepo.On("GetAll", mock.AnythingOfType("model.QueryOptions")).Return(model.MediaFiles{song1, song2}, nil).Once()
		mediaFileRepo.On("GetAll", mock.AnythingOfType("model.QueryOptions")).Return(model.MediaFiles{song3}, nil).Once()

		songs, err := provider.ArtistRadio(ctx, "artist-1", 3)

		Expect(err).ToNot(HaveOccurred())
		Expect(songs).To(HaveLen(3))
		for _, song := range songs {
			Expect(song.ID).To(BeElementOf("song-1", "song-2", "song-3"))
		}
	})

	It("returns ErrNotFound when artist is not found", func() {
		artistRepo.On("Get", "artist-unknown-artist").Return(nil, model.ErrNotFound)
		mediaFileRepo.On("Get", "artist-unknown-artist").Return(nil, model.ErrNotFound)

		artistRepo.On("GetAll", mock.MatchedBy(func(opt model.QueryOptions) bool {
			return opt.Max == 1 && opt.Filters != nil
		})).Return(model.Artists{}, nil).Maybe()

		songs, err := provider.ArtistRadio(ctx, "artist-unknown-artist", 5)

		Expect(err).To(Equal(model.ErrNotFound))
		Expect(songs).To(BeNil())
	})

	It("returns songs from main artist when GetSimilarArtists returns error", func() {
		artist1 := model.Artist{ID: "artist-1", Name: "Artist One"}
		song1 := model.MediaFile{ID: "song-1", Title: "Song One", ArtistID: "artist-1", MbzRecordingID: "mbid-1"}

		artistRepo.On("Get", "artist-1").Return(&artist1, nil).Maybe()
		artistRepo.On("GetAll", mock.MatchedBy(func(opt model.QueryOptions) bool {
			return opt.Max == 1 && opt.Filters != nil
		})).Return(model.Artists{artist1}, nil).Maybe()

		mockAgent.On("GetSimilarArtists", mock.Anything, "artist-1", "Artist One", "", 15).
			Return(nil, errors.New("error getting similar artists")).Once()

		artistRepo.On("GetAll", mock.MatchedBy(func(opt model.QueryOptions) bool {
			return opt.Max == 0 && opt.Filters != nil
		})).Return(model.Artists{}, nil).Once()

		mockAgent.On("GetArtistTopSongs", mock.Anything, "artist-1", "Artist One", "", mock.Anything).
			Return([]agents.Song{
				{Name: "Song One", MBID: "mbid-1"},
			}, nil).Once()

		mediaFileRepo.On("GetAll", mock.AnythingOfType("model.QueryOptions")).Return(model.MediaFiles{song1}, nil).Once()

		songs, err := provider.ArtistRadio(ctx, "artist-1", 5)

		Expect(err).ToNot(HaveOccurred())
		Expect(songs).To(HaveLen(1))
		Expect(songs[0].ID).To(Equal("song-1"))
	})

	It("returns empty list when GetArtistTopSongs returns error", func() {
		artist1 := model.Artist{ID: "artist-1", Name: "Artist One"}

		artistRepo.On("Get", "artist-1").Return(&artist1, nil).Maybe()
		artistRepo.On("GetAll", mock.MatchedBy(func(opt model.QueryOptions) bool {
			return opt.Max == 1 && opt.Filters != nil
		})).Return(model.Artists{artist1}, nil).Maybe()

		mockAgent.On("GetSimilarArtists", mock.Anything, "artist-1", "Artist One", "", 15).
			Return([]agents.Artist{}, nil).Once()

		artistRepo.On("GetAll", mock.MatchedBy(func(opt model.QueryOptions) bool {
			return opt.Max == 0 && opt.Filters != nil
		})).Return(model.Artists{}, nil).Once()

		mockAgent.On("GetArtistTopSongs", mock.Anything, "artist-1", "Artist One", "", mock.Anything).
			Return(nil, errors.New("error getting top songs")).Once()

		songs, err := provider.ArtistRadio(ctx, "artist-1", 5)

		Expect(err).ToNot(HaveOccurred())
		Expect(songs).To(BeEmpty())
	})

	It("respects count parameter", func() {
		artist1 := model.Artist{ID: "artist-1", Name: "Artist One"}
		song1 := model.MediaFile{ID: "song-1", Title: "Song One", ArtistID: "artist-1", MbzRecordingID: "mbid-1"}
		song2 := model.MediaFile{ID: "song-2", Title: "Song Two", ArtistID: "artist-1", MbzRecordingID: "mbid-2"}

		artistRepo.On("Get", "artist-1").Return(&artist1, nil).Maybe()
		artistRepo.On("GetAll", mock.MatchedBy(func(opt model.QueryOptions) bool {
			return opt.Max == 1 && opt.Filters != nil
		})).Return(model.Artists{artist1}, nil).Maybe()

		mockAgent.On("GetSimilarArtists", mock.Anything, "artist-1", "Artist One", "", 15).
			Return([]agents.Artist{}, nil).Once()

		artistRepo.On("GetAll", mock.MatchedBy(func(opt model.QueryOptions) bool {
			return opt.Max == 0 && opt.Filters != nil
		})).Return(model.Artists{}, nil).Once()

		mockAgent.On("GetArtistTopSongs", mock.Anything, "artist-1", "Artist One", "", mock.Anything).
			Return([]agents.Song{
				{Name: "Song One", MBID: "mbid-1"},
				{Name: "Song Two", MBID: "mbid-2"},
			}, nil).Once()

		mediaFileRepo.On("GetAll", mock.AnythingOfType("model.QueryOptions")).Return(model.MediaFiles{song1, song2}, nil).Once()

		songs, err := provider.ArtistRadio(ctx, "artist-1", 1)

		Expect(err).ToNot(HaveOccurred())
		Expect(songs).To(HaveLen(1))
		Expect(songs[0].ID).To(BeElementOf("song-1", "song-2"))
	})
})
