package agents

import (
	"context"
	"errors"

	"github.com/navidrome/navidrome/conf/configtest"
	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/tests"

	"github.com/navidrome/navidrome/conf"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Agents", func() {
	var ctx context.Context
	var cancel context.CancelFunc
	var ds model.DataStore
	var mfRepo *tests.MockMediaFileRepo
	BeforeEach(func() {
		DeferCleanup(configtest.SetupConfig())
		ctx, cancel = context.WithCancel(context.Background())
		mfRepo = tests.CreateMockMediaFileRepo()
		ds = &tests.MockDataStore{MockedMediaFile: mfRepo}
	})

	Describe("Local", func() {
		var ag *Agents
		BeforeEach(func() {
			conf.Server.Agents = ""
			ag = createAgents(ds, nil)
		})

		It("calls the placeholder GetArtistImages", func() {
			mfRepo.SetData(model.MediaFiles{{ID: "1", Title: "One", MbzReleaseTrackID: "111"}, {ID: "2", Title: "Two", MbzReleaseTrackID: "222"}})
			songs, err := ag.GetArtistTopSongs(ctx, "123", "John Doe", "mb123", 2)
			Expect(err).ToNot(HaveOccurred())
			Expect(songs).To(ConsistOf([]Song{{Name: "One", MBID: "111"}, {Name: "Two", MBID: "222"}}))
		})
	})

	Describe("Agents", func() {
		var ag *Agents
		var mock *mockAgent
		BeforeEach(func() {
			mock = &mockAgent{}
			Register("fake", func(model.DataStore) Interface { return mock })
			Register("disabled", func(model.DataStore) Interface { return nil })
			Register("empty", func(model.DataStore) Interface { return &emptyAgent{} })
			conf.Server.Agents = "empty,fake,disabled"
			ag = createAgents(ds, nil)
			Expect(ag.AgentName()).To(Equal("agents"))
		})

		It("does not register disabled agents", func() {
			var ags []string
			for _, enabledAgent := range ag.getEnabledAgentNames() {
				agent := ag.getAgent(enabledAgent)
				if agent != nil {
					ags = append(ags, agent.AgentName())
				}
			}
			// local agent is always appended to the end of the agents list
			Expect(ags).To(HaveExactElements("empty", "fake", "local"))
			Expect(ags).ToNot(ContainElement("disabled"))
		})

		Describe("GetArtistMBID", func() {
			It("returns on first match", func() {
				Expect(ag.GetArtistMBID(ctx, "123", "test")).To(Equal("mbid"))
				Expect(mock.Args).To(HaveExactElements("123", "test"))
			})
			It("returns empty if artist is Various Artists", func() {
				mbid, err := ag.GetArtistMBID(ctx, consts.VariousArtistsID, consts.VariousArtists)
				Expect(err).ToNot(HaveOccurred())
				Expect(mbid).To(BeEmpty())
				Expect(mock.Args).To(BeEmpty())
			})
			It("returns not found if artist is Unknown Artist", func() {
				mbid, err := ag.GetArtistMBID(ctx, consts.VariousArtistsID, consts.VariousArtists)
				Expect(err).ToNot(HaveOccurred())
				Expect(mbid).To(BeEmpty())
				Expect(mock.Args).To(BeEmpty())
			})
			It("skips the agent if it returns an error", func() {
				mock.Err = errors.New("error")
				_, err := ag.GetArtistMBID(ctx, "123", "test")
				Expect(err).To(MatchError(ErrNotFound))
				Expect(mock.Args).To(HaveExactElements("123", "test"))
			})
			It("interrupts if the context is canceled", func() {
				cancel()
				_, err := ag.GetArtistMBID(ctx, "123", "test")
				Expect(err).To(MatchError(ErrNotFound))
				Expect(mock.Args).To(BeEmpty())
			})
		})

		Describe("GetArtistURL", func() {
			It("returns on first match", func() {
				Expect(ag.GetArtistURL(ctx, "123", "test", "mb123")).To(Equal("url"))
				Expect(mock.Args).To(HaveExactElements("123", "test", "mb123"))
			})
			It("returns empty if artist is Various Artists", func() {
				url, err := ag.GetArtistURL(ctx, consts.VariousArtistsID, consts.VariousArtists, "")
				Expect(err).ToNot(HaveOccurred())
				Expect(url).To(BeEmpty())
				Expect(mock.Args).To(BeEmpty())
			})
			It("returns not found if artist is Unknown Artist", func() {
				url, err := ag.GetArtistURL(ctx, consts.VariousArtistsID, consts.VariousArtists, "")
				Expect(err).ToNot(HaveOccurred())
				Expect(url).To(BeEmpty())
				Expect(mock.Args).To(BeEmpty())
			})
			It("skips the agent if it returns an error", func() {
				mock.Err = errors.New("error")
				_, err := ag.GetArtistURL(ctx, "123", "test", "mb123")
				Expect(err).To(MatchError(ErrNotFound))
				Expect(mock.Args).To(HaveExactElements("123", "test", "mb123"))
			})
			It("interrupts if the context is canceled", func() {
				cancel()
				_, err := ag.GetArtistURL(ctx, "123", "test", "mb123")
				Expect(err).To(MatchError(ErrNotFound))
				Expect(mock.Args).To(BeEmpty())
			})
		})

		Describe("GetArtistBiography", func() {
			It("returns on first match", func() {
				Expect(ag.GetArtistBiography(ctx, "123", "test", "mb123")).To(Equal("bio"))
				Expect(mock.Args).To(HaveExactElements("123", "test", "mb123"))
			})
			It("returns empty if artist is Various Artists", func() {
				bio, err := ag.GetArtistBiography(ctx, consts.VariousArtistsID, consts.VariousArtists, "")
				Expect(err).ToNot(HaveOccurred())
				Expect(bio).To(BeEmpty())
				Expect(mock.Args).To(BeEmpty())
			})
			It("returns not found if artist is Unknown Artist", func() {
				bio, err := ag.GetArtistBiography(ctx, consts.VariousArtistsID, consts.VariousArtists, "")
				Expect(err).ToNot(HaveOccurred())
				Expect(bio).To(BeEmpty())
				Expect(mock.Args).To(BeEmpty())
			})
			It("skips the agent if it returns an error", func() {
				mock.Err = errors.New("error")
				_, err := ag.GetArtistBiography(ctx, "123", "test", "mb123")
				Expect(err).To(MatchError(ErrNotFound))
				Expect(mock.Args).To(HaveExactElements("123", "test", "mb123"))
			})
			It("interrupts if the context is canceled", func() {
				cancel()
				_, err := ag.GetArtistBiography(ctx, "123", "test", "mb123")
				Expect(err).To(MatchError(ErrNotFound))
				Expect(mock.Args).To(BeEmpty())
			})
		})

		Describe("GetArtistImages", func() {
			It("returns on first match", func() {
				Expect(ag.GetArtistImages(ctx, "123", "test", "mb123")).To(Equal([]ExternalImage{{
					URL:  "imageUrl",
					Size: 100,
				}}))
				Expect(mock.Args).To(HaveExactElements("123", "test", "mb123"))
			})
			It("skips the agent if it returns an error", func() {
				mock.Err = errors.New("error")
				_, err := ag.GetArtistImages(ctx, "123", "test", "mb123")
				Expect(err).To(MatchError("not found"))
				Expect(mock.Args).To(HaveExactElements("123", "test", "mb123"))
			})
			It("interrupts if the context is canceled", func() {
				cancel()
				_, err := ag.GetArtistImages(ctx, "123", "test", "mb123")
				Expect(err).To(MatchError(ErrNotFound))
				Expect(mock.Args).To(BeEmpty())
			})

			Context("with multiple image agents", func() {
				var first *testImageAgent
				var second *testImageAgent

				BeforeEach(func() {
					first = &testImageAgent{Name: "imgFail", Err: errors.New("fail")}
					second = &testImageAgent{Name: "imgOk", Images: []ExternalImage{{URL: "ok", Size: 1}}}
					Register("imgFail", func(model.DataStore) Interface { return first })
					Register("imgOk", func(model.DataStore) Interface { return second })
				})

				It("falls back to the next agent on error", func() {
					conf.Server.Agents = "imgFail,imgOk"
					ag = createAgents(ds, nil)

					images, err := ag.GetArtistImages(ctx, "id", "artist", "mbid")
					Expect(err).ToNot(HaveOccurred())
					Expect(images).To(Equal([]ExternalImage{{URL: "ok", Size: 1}}))
					Expect(first.Args).To(HaveExactElements("id", "artist", "mbid"))
					Expect(second.Args).To(HaveExactElements("id", "artist", "mbid"))
				})

				It("falls back if the first agent returns no images", func() {
					first.Err = nil
					first.Images = []ExternalImage{}
					conf.Server.Agents = "imgFail,imgOk"
					ag = createAgents(ds, nil)

					images, err := ag.GetArtistImages(ctx, "id", "artist", "mbid")
					Expect(err).ToNot(HaveOccurred())
					Expect(images).To(Equal([]ExternalImage{{URL: "ok", Size: 1}}))
					Expect(first.Args).To(HaveExactElements("id", "artist", "mbid"))
					Expect(second.Args).To(HaveExactElements("id", "artist", "mbid"))
				})
			})
		})

		Describe("GetSimilarArtists", func() {
			It("returns on first match", func() {
				Expect(ag.GetSimilarArtists(ctx, "123", "test", "mb123", 1)).To(Equal([]Artist{{
					Name: "Joe Dohn",
					MBID: "mbid321",
				}}))
				Expect(mock.Args).To(HaveExactElements("123", "test", "mb123", 1))
			})
			It("skips the agent if it returns an error", func() {
				mock.Err = errors.New("error")
				_, err := ag.GetSimilarArtists(ctx, "123", "test", "mb123", 1)
				Expect(err).To(MatchError(ErrNotFound))
				Expect(mock.Args).To(HaveExactElements("123", "test", "mb123", 1))
			})
			It("interrupts if the context is canceled", func() {
				cancel()
				_, err := ag.GetSimilarArtists(ctx, "123", "test", "mb123", 1)
				Expect(err).To(MatchError(ErrNotFound))
				Expect(mock.Args).To(BeEmpty())
			})
		})

		Describe("GetArtistTopSongs", func() {
			It("returns on first match", func() {
				conf.Server.DevExternalArtistFetchMultiplier = 1
				Expect(ag.GetArtistTopSongs(ctx, "123", "test", "mb123", 2)).To(Equal([]Song{{
					Name: "A Song",
					MBID: "mbid444",
				}}))
				Expect(mock.Args).To(HaveExactElements("123", "test", "mb123", 2))
			})
			It("skips the agent if it returns an error", func() {
				conf.Server.DevExternalArtistFetchMultiplier = 1
				mock.Err = errors.New("error")
				_, err := ag.GetArtistTopSongs(ctx, "123", "test", "mb123", 2)
				Expect(err).To(MatchError(ErrNotFound))
				Expect(mock.Args).To(HaveExactElements("123", "test", "mb123", 2))
			})
			It("interrupts if the context is canceled", func() {
				cancel()
				_, err := ag.GetArtistTopSongs(ctx, "123", "test", "mb123", 2)
				Expect(err).To(MatchError(ErrNotFound))
				Expect(mock.Args).To(BeEmpty())
			})
			It("fetches with multiplier", func() {
				conf.Server.DevExternalArtistFetchMultiplier = 2
				Expect(ag.GetArtistTopSongs(ctx, "123", "test", "mb123", 2)).To(Equal([]Song{{
					Name: "A Song",
					MBID: "mbid444",
				}}))
				Expect(mock.Args).To(HaveExactElements("123", "test", "mb123", 4))
			})
		})

		Describe("GetAlbumInfo", func() {
			It("returns meaningful data", func() {
				Expect(ag.GetAlbumInfo(ctx, "album", "artist", "mbid")).To(Equal(&AlbumInfo{
					Name:        "A Song",
					MBID:        "mbid444",
					Description: "A Description",
					URL:         "External URL",
				}))
				Expect(mock.Args).To(HaveExactElements("album", "artist", "mbid"))
			})
			It("skips the agent if it returns an error", func() {
				mock.Err = errors.New("error")
				_, err := ag.GetAlbumInfo(ctx, "album", "artist", "mbid")
				Expect(err).To(MatchError(ErrNotFound))
				Expect(mock.Args).To(HaveExactElements("album", "artist", "mbid"))
			})
			It("interrupts if the context is canceled", func() {
				cancel()
				_, err := ag.GetAlbumInfo(ctx, "album", "artist", "mbid")
				Expect(err).To(MatchError(ErrNotFound))
				Expect(mock.Args).To(BeEmpty())
			})
		})
	})
})

type mockAgent struct {
	Args []interface{}
	Err  error
}

func (a *mockAgent) AgentName() string {
	return "fake"
}

func (a *mockAgent) GetArtistMBID(_ context.Context, id string, name string) (string, error) {
	a.Args = []interface{}{id, name}
	if a.Err != nil {
		return "", a.Err
	}
	return "mbid", nil
}

func (a *mockAgent) GetArtistURL(_ context.Context, id, name, mbid string) (string, error) {
	a.Args = []interface{}{id, name, mbid}
	if a.Err != nil {
		return "", a.Err
	}
	return "url", nil
}

func (a *mockAgent) GetArtistBiography(_ context.Context, id, name, mbid string) (string, error) {
	a.Args = []interface{}{id, name, mbid}
	if a.Err != nil {
		return "", a.Err
	}
	return "bio", nil
}

func (a *mockAgent) GetArtistImages(_ context.Context, id, name, mbid string) ([]ExternalImage, error) {
	a.Args = []interface{}{id, name, mbid}
	if a.Err != nil {
		return nil, a.Err
	}
	return []ExternalImage{{
		URL:  "imageUrl",
		Size: 100,
	}}, nil
}

func (a *mockAgent) GetSimilarArtists(_ context.Context, id, name, mbid string, limit int) ([]Artist, error) {
	a.Args = []interface{}{id, name, mbid, limit}
	if a.Err != nil {
		return nil, a.Err
	}
	return []Artist{{
		Name: "Joe Dohn",
		MBID: "mbid321",
	}}, nil
}

func (a *mockAgent) GetArtistTopSongs(_ context.Context, id, artistName, mbid string, count int) ([]Song, error) {
	a.Args = []interface{}{id, artistName, mbid, count}
	if a.Err != nil {
		return nil, a.Err
	}
	return []Song{{
		Name: "A Song",
		MBID: "mbid444",
	}}, nil
}

func (a *mockAgent) GetAlbumInfo(ctx context.Context, name, artist, mbid string) (*AlbumInfo, error) {
	a.Args = []interface{}{name, artist, mbid}
	if a.Err != nil {
		return nil, a.Err
	}
	return &AlbumInfo{
		Name:        "A Song",
		MBID:        "mbid444",
		Description: "A Description",
		URL:         "External URL",
	}, nil
}

type emptyAgent struct {
	Interface
}

func (e *emptyAgent) AgentName() string {
	return "empty"
}

type testImageAgent struct {
	Name   string
	Images []ExternalImage
	Err    error
	Args   []interface{}
}

func (t *testImageAgent) AgentName() string { return t.Name }

func (t *testImageAgent) GetArtistImages(_ context.Context, id, name, mbid string) ([]ExternalImage, error) {
	t.Args = []interface{}{id, name, mbid}
	return t.Images, t.Err
}
