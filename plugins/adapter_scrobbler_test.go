package plugins

import (
	"context"
	"errors"

	"github.com/navidrome/navidrome/core/scrobbler"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/plugins/api"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

type mockScrobblerService struct {
	isAuthorizedResp *api.ScrobblerIsAuthorizedResponse
	isAuthorizedErr  error
	nowPlayingResp   *api.ScrobblerNowPlayingResponse
	nowPlayingErr    error
	scrobbleResp     *api.ScrobblerScrobbleResponse
	scrobbleErr      error
}

func (m *mockScrobblerService) IsAuthorized(ctx context.Context, req *api.ScrobblerIsAuthorizedRequest) (*api.ScrobblerIsAuthorizedResponse, error) {
	return m.isAuthorizedResp, m.isAuthorizedErr
}
func (m *mockScrobblerService) NowPlaying(ctx context.Context, req *api.ScrobblerNowPlayingRequest) (*api.ScrobblerNowPlayingResponse, error) {
	return m.nowPlayingResp, m.nowPlayingErr
}
func (m *mockScrobblerService) Scrobble(ctx context.Context, req *api.ScrobblerScrobbleRequest) (*api.ScrobblerScrobbleResponse, error) {
	return m.scrobbleResp, m.scrobbleErr
}

// Mock loader for tests
type mockLoader struct {
	mockService *mockScrobblerService
}

// Define a test-specific version of wasmScrobblerPlugin that uses mockLoader
type testWasmScrobblerPlugin struct {
	*wasmBasePlugin[api.ScrobblerService, *mockLoader]
}

// Implement the same methods as wasmScrobblerPlugin
func (w *testWasmScrobblerPlugin) PluginName() string {
	return w.name
}

func (w *testWasmScrobblerPlugin) IsAuthorized(ctx context.Context, userId string) bool {
	result, err := callMethod(ctx, w, "IsAuthorized", func(inst api.ScrobblerService) (bool, error) {
		resp, err := inst.IsAuthorized(ctx, &api.ScrobblerIsAuthorizedRequest{UserId: userId})
		if err != nil {
			return false, err
		}
		if resp.Error != "" {
			return false, nil
		}
		return resp.Authorized, nil
	})
	return err == nil && result
}

func (w *testWasmScrobblerPlugin) NowPlaying(ctx context.Context, userId string, track *model.MediaFile) error {
	artists := make([]*api.Artist, 0, len(track.Participants[model.RoleArtist]))
	for _, a := range track.Participants[model.RoleArtist] {
		artists = append(artists, &api.Artist{Name: a.Name, Mbid: a.MbzArtistID})
	}
	albumArtists := make([]*api.Artist, 0, len(track.Participants[model.RoleAlbumArtist]))
	for _, a := range track.Participants[model.RoleAlbumArtist] {
		albumArtists = append(albumArtists, &api.Artist{Name: a.Name, Mbid: a.MbzArtistID})
	}
	trackInfo := &api.TrackInfo{
		Id:           track.ID,
		Mbid:         track.MbzRecordingID,
		Name:         track.Title,
		Album:        track.Album,
		AlbumMbid:    track.MbzAlbumID,
		Artists:      artists,
		AlbumArtists: albumArtists,
		Length:       int32(track.Duration),
	}
	_, err := callMethod(ctx, w, "NowPlaying", func(inst api.ScrobblerService) (struct{}, error) {
		resp, err := inst.NowPlaying(ctx, &api.ScrobblerNowPlayingRequest{
			UserId: userId,
			Track:  trackInfo,
		})
		if err != nil {
			return struct{}{}, err
		}
		if resp.Error != "" {
			return struct{}{}, nil
		}
		return struct{}{}, nil
	})
	return err
}

func (w *testWasmScrobblerPlugin) Scrobble(ctx context.Context, userId string, s scrobbler.Scrobble) error {
	track := &s.MediaFile
	artists := make([]*api.Artist, 0, len(track.Participants[model.RoleArtist]))
	for _, a := range track.Participants[model.RoleArtist] {
		artists = append(artists, &api.Artist{Name: a.Name, Mbid: a.MbzArtistID})
	}
	albumArtists := make([]*api.Artist, 0, len(track.Participants[model.RoleAlbumArtist]))
	for _, a := range track.Participants[model.RoleAlbumArtist] {
		albumArtists = append(albumArtists, &api.Artist{Name: a.Name, Mbid: a.MbzArtistID})
	}
	trackInfo := &api.TrackInfo{
		Id:           track.ID,
		Mbid:         track.MbzRecordingID,
		Name:         track.Title,
		Album:        track.Album,
		AlbumMbid:    track.MbzAlbumID,
		Artists:      artists,
		AlbumArtists: albumArtists,
		Length:       int32(track.Duration),
	}
	_, err := callMethod(ctx, w, "Scrobble", func(inst api.ScrobblerService) (struct{}, error) {
		resp, err := inst.Scrobble(ctx, &api.ScrobblerScrobbleRequest{
			UserId:    userId,
			Track:     trackInfo,
			Timestamp: s.TimeStamp.Unix(),
		})
		if err != nil {
			return struct{}{}, err
		}
		if resp.Error != "" {
			return struct{}{}, nil
		}
		return struct{}{}, nil
	})
	return err
}

var _ = Describe("wasmScrobblerPlugin", func() {
	var (
		ctx    context.Context
		plugin *testWasmScrobblerPlugin
		mock   *mockScrobblerService
	)

	BeforeEach(func() {
		ctx = context.Background()
		mock = &mockScrobblerService{}
		plugin = &testWasmScrobblerPlugin{
			wasmBasePlugin: &wasmBasePlugin[api.ScrobblerService, *mockLoader]{
				name:   "test-plugin",
				loader: &mockLoader{mockService: mock},
				loadFunc: func(_ context.Context, l *mockLoader, _ string) (api.ScrobblerService, error) {
					return l.mockService, nil
				},
			},
		}
	})

	It("returns true for IsAuthorized when plugin says authorized", func() {
		mock.isAuthorizedResp = &api.ScrobblerIsAuthorizedResponse{Authorized: true}
		Expect(plugin.IsAuthorized(ctx, "user1")).To(BeTrue())
	})

	It("returns false for IsAuthorized when plugin says not authorized", func() {
		mock.isAuthorizedResp = &api.ScrobblerIsAuthorizedResponse{Authorized: false}
		Expect(plugin.IsAuthorized(ctx, "user1")).To(BeFalse())
	})

	It("returns false for IsAuthorized on error", func() {
		mock.isAuthorizedErr = errors.New("fail")
		Expect(plugin.IsAuthorized(ctx, "user1")).To(BeFalse())
	})

	It("calls NowPlaying and returns no error", func() {
		mock.nowPlayingResp = &api.ScrobblerNowPlayingResponse{}
		track := &model.MediaFile{ID: "t1", Title: "Song", Album: "Album", Duration: 123}
		Expect(plugin.NowPlaying(ctx, "user1", track)).To(Succeed())
	})

	It("calls NowPlaying and returns error", func() {
		mock.nowPlayingErr = errors.New("fail")
		track := &model.MediaFile{ID: "t1", Title: "Song", Album: "Album", Duration: 123}
		Expect(plugin.NowPlaying(ctx, "user1", track)).ToNot(Succeed())
	})

	It("calls Scrobble and returns no error", func() {
		mock.scrobbleResp = &api.ScrobblerScrobbleResponse{}
		s := scrobbler.Scrobble{MediaFile: model.MediaFile{ID: "t1", Title: "Song", Album: "Album", Duration: 123}}
		Expect(plugin.Scrobble(ctx, "user1", s)).To(Succeed())
	})

	It("calls Scrobble and returns error", func() {
		mock.scrobbleErr = errors.New("fail")
		s := scrobbler.Scrobble{MediaFile: model.MediaFile{ID: "t1", Title: "Song", Album: "Album", Duration: 123}}
		Expect(plugin.Scrobble(ctx, "user1", s)).ToNot(Succeed())
	})
})
