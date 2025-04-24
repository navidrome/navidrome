package plugins

import (
	"context"
	"errors"
	"testing"

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

var _ = Describe("wasmScrobblerPlugin", func() {
	var (
		ctx    context.Context
		plugin *wasmScrobblerPlugin
		mock   *mockScrobblerService
	)

	BeforeEach(func() {
		ctx = context.Background()
		mock = &mockScrobblerService{}
		plugin = &wasmScrobblerPlugin{
			wasmBasePlugin: &wasmBasePlugin[api.ScrobblerService]{
				name: "test-plugin",
				loadFunc: func(_ context.Context, _ any, _ string) (api.ScrobblerService, error) {
					return mock, nil
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

func TestScrobblerPlugin(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "ScrobblerPlugin Suite")
}
