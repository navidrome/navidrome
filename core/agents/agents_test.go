package agents

import (
	"context"
	"errors"

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
		ctx, cancel = context.WithCancel(context.Background())
		mfRepo = tests.CreateMockMediaFileRepo()
		ds = &tests.MockDataStore{MockedMediaFile: mfRepo}
	})

	Describe("Local", func() {
		var ag *Agents
		BeforeEach(func() {
			conf.Server.Agents = ""
			ag = New(ds)
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
			Register("fake", func(ds model.DataStore) Interface {
				return mock
			})
			Register("empty", func(ds model.DataStore) Interface {
				return struct {
					Interface
				}{}
			})
			conf.Server.Agents = "empty,fake"
			ag = New(ds)
			Expect(ag.AgentName()).To(Equal("agents"))
		})

		Describe("GetArtistMBID", func() {
			It("returns on first match", func() {
				Expect(ag.GetArtistMBID(ctx, "123", "test")).To(Equal("mbid"))
				Expect(mock.Args).To(ConsistOf("123", "test"))
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
				Expect(mock.Args).To(ConsistOf("123", "test"))
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
				Expect(mock.Args).To(ConsistOf("123", "test", "mb123"))
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
				Expect(mock.Args).To(ConsistOf("123", "test", "mb123"))
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
				Expect(mock.Args).To(ConsistOf("123", "test", "mb123"))
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
				Expect(mock.Args).To(ConsistOf("123", "test", "mb123"))
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
				Expect(mock.Args).To(ConsistOf("123", "test", "mb123"))
			})
			It("skips the agent if it returns an error", func() {
				mock.Err = errors.New("error")
				_, err := ag.GetArtistImages(ctx, "123", "test", "mb123")
				Expect(err).To(MatchError("not found"))
				Expect(mock.Args).To(ConsistOf("123", "test", "mb123"))
			})
			It("interrupts if the context is canceled", func() {
				cancel()
				_, err := ag.GetArtistImages(ctx, "123", "test", "mb123")
				Expect(err).To(MatchError(ErrNotFound))
				Expect(mock.Args).To(BeEmpty())
			})
		})

		Describe("GetSimilarArtists", func() {
			It("returns on first match", func() {
				Expect(ag.GetSimilarArtists(ctx, "123", "test", "mb123", 1)).To(Equal([]Artist{{
					Name: "Joe Dohn",
					MBID: "mbid321",
				}}))
				Expect(mock.Args).To(ConsistOf("123", "test", "mb123", 1))
			})
			It("skips the agent if it returns an error", func() {
				mock.Err = errors.New("error")
				_, err := ag.GetSimilarArtists(ctx, "123", "test", "mb123", 1)
				Expect(err).To(MatchError(ErrNotFound))
				Expect(mock.Args).To(ConsistOf("123", "test", "mb123", 1))
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
				Expect(ag.GetArtistTopSongs(ctx, "123", "test", "mb123", 2)).To(Equal([]Song{{
					Name: "A Song",
					MBID: "mbid444",
				}}))
				Expect(mock.Args).To(ConsistOf("123", "test", "mb123", 2))
			})
			It("skips the agent if it returns an error", func() {
				mock.Err = errors.New("error")
				_, err := ag.GetArtistTopSongs(ctx, "123", "test", "mb123", 2)
				Expect(err).To(MatchError(ErrNotFound))
				Expect(mock.Args).To(ConsistOf("123", "test", "mb123", 2))
			})
			It("interrupts if the context is canceled", func() {
				cancel()
				_, err := ag.GetArtistTopSongs(ctx, "123", "test", "mb123", 2)
				Expect(err).To(MatchError(ErrNotFound))
				Expect(mock.Args).To(BeEmpty())
			})
		})

		Describe("GetAlbumInfo", func() {
			It("returns meaningful data", func() {
				Expect(ag.GetAlbumInfo(ctx, "album", "artist", "mbid")).To(Equal(&AlbumInfo{
					Name:        "A Song",
					MBID:        "mbid444",
					Description: "A Description",
					URL:         "External URL",
					Images: []ExternalImage{
						{
							Size: 174,
							URL:  "https://lastfm.freetls.fastly.net/i/u/174s/00000000000000000000000000000000.png",
						}, {
							Size: 64,
							URL:  "https://lastfm.freetls.fastly.net/i/u/64s/00000000000000000000000000000000.png",
						}, {
							Size: 34,
							URL:  "https://lastfm.freetls.fastly.net/i/u/34s/00000000000000000000000000000000.png",
						},
					},
				}))
				Expect(mock.Args).To(ConsistOf("album", "artist", "mbid"))
			})
			It("skips the agent if it returns an error", func() {
				mock.Err = errors.New("error")
				_, err := ag.GetAlbumInfo(ctx, "album", "artist", "mbid")
				Expect(err).To(MatchError(ErrNotFound))
				Expect(mock.Args).To(ConsistOf("album", "artist", "mbid"))
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
		Images: []ExternalImage{
			{
				Size: 174,
				URL:  "https://lastfm.freetls.fastly.net/i/u/174s/00000000000000000000000000000000.png",
			}, {
				Size: 64,
				URL:  "https://lastfm.freetls.fastly.net/i/u/64s/00000000000000000000000000000000.png",
			}, {
				Size: 34,
				URL:  "https://lastfm.freetls.fastly.net/i/u/34s/00000000000000000000000000000000.png",
			},
		},
	}, nil
}
