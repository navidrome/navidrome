package sonic_test

import (
	"context"
	"errors"

	"github.com/navidrome/navidrome/core/agents"
	"github.com/navidrome/navidrome/core/matcher"
	"github.com/navidrome/navidrome/core/sonic"
	"github.com/navidrome/navidrome/db"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/persistence"
	"github.com/navidrome/navidrome/persistence/persistencetest"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

type mockPluginLoader struct {
	names    []string
	provider sonic.Provider
	loadOk   bool
}

func (m *mockPluginLoader) PluginNames(capability string) []string {
	if capability == "SonicSimilarity" {
		return m.names
	}
	return nil
}

func (m *mockPluginLoader) LoadSonicSimilarity(name string) (sonic.Provider, bool) {
	return m.provider, m.loadOk
}

type mockProvider struct {
	similarResults []sonic.SimilarResult
	similarErr     error
	pathResults    []sonic.SimilarResult
	pathErr        error
}

func (m *mockProvider) GetSonicSimilarTracks(_ context.Context, _ *model.MediaFile, _ int) ([]sonic.SimilarResult, error) {
	return m.similarResults, m.similarErr
}

func (m *mockProvider) FindSonicPath(_ context.Context, _, _ *model.MediaFile, _ int) ([]sonic.SimilarResult, error) {
	return m.pathResults, m.pathErr
}

var _ = Describe("Sonic", func() {
	var (
		ctx     context.Context
		ds      model.DataStore
		loader  *mockPluginLoader
		service *sonic.Sonic
	)

	BeforeEach(func() {
		ctx = GinkgoT().Context()
		ds = persistence.New(db.Db())
		loader = &mockPluginLoader{}
	})

	Describe("HasProvider", func() {
		It("returns false when no plugins available", func() {
			loader.names = nil
			service = sonic.New(ds, loader, nil)
			Expect(service.HasProvider()).To(BeFalse())
		})

		It("returns true when a plugin is available", func() {
			loader.names = []string{"test-plugin"}
			service = sonic.New(ds, loader, nil)
			Expect(service.HasProvider()).To(BeTrue())
		})
	})

	Describe("GetSonicSimilarTracks", func() {
		It("returns error when no plugin available", func() {
			loader.names = nil
			service = sonic.New(ds, loader, nil)
			_, err := service.GetSonicSimilarTracks(ctx, "song-1", 10)
			Expect(err).To(MatchError(model.ErrNotFound))
		})

		It("returns error when media file not found", func() {
			loader.names = []string{"test-plugin"}
			loader.provider = &mockProvider{}
			loader.loadOk = true
			service = sonic.New(ds, loader, matcher.New(ds))
			_, err := service.GetSonicSimilarTracks(ctx, "nonexistent", 10)
			Expect(err).To(MatchError(model.ErrNotFound))
		})

		It("returns matched results from plugin", func() {
			persistencetest.PutMediaFiles(ctx, ds,
				model.MediaFile{ID: "song-1", Title: "Test Song", Artist: "Test Artist"},
				model.MediaFile{ID: "song-2", Title: "Similar Song", Artist: "Test Artist"},
			)

			provider := &mockProvider{
				similarResults: []sonic.SimilarResult{
					{Song: agents.Song{ID: "song-2", Name: "Similar Song", Artists: []agents.Artist{{Name: "Test Artist"}}}, Similarity: 0.85},
				},
			}
			loader.names = []string{"test-plugin"}
			loader.provider = provider
			loader.loadOk = true

			service = sonic.New(ds, loader, matcher.New(ds))
			matches, err := service.GetSonicSimilarTracks(ctx, "song-1", 10)
			Expect(err).ToNot(HaveOccurred())
			Expect(matches).To(HaveLen(1))
			Expect(matches[0].MediaFile.ID).To(Equal("song-2"))
			Expect(matches[0].Similarity).To(Equal(0.85))
		})
	})

	Describe("FindSonicPath", func() {
		It("returns error when no plugin available", func() {
			loader.names = nil
			service = sonic.New(ds, loader, nil)
			_, err := service.FindSonicPath(ctx, "song-1", "song-2", 25)
			Expect(err).To(MatchError(model.ErrNotFound))
		})

		It("returns error when plugin call fails", func() {
			persistencetest.PutMediaFiles(ctx, ds,
				model.MediaFile{ID: "song-1", Title: "Start", Artist: "Artist"},
				model.MediaFile{ID: "song-2", Title: "End", Artist: "Artist"},
			)

			provider := &mockProvider{pathErr: errors.New("plugin error")}
			loader.names = []string{"test-plugin"}
			loader.provider = provider
			loader.loadOk = true

			service = sonic.New(ds, loader, matcher.New(ds))
			_, err := service.FindSonicPath(ctx, "song-1", "song-2", 25)
			Expect(err).To(HaveOccurred())
		})
	})
})
