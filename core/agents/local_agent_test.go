package agents

import (
	"context"

	"github.com/navidrome/navidrome/db"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/persistence"
	"github.com/navidrome/navidrome/persistence/persistencetest"
	"github.com/navidrome/navidrome/utils/slice"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("localAgent GetSimilarSongsByTrack", func() {
	var ds *getRandomSpy
	var agent *localAgent
	var ctx context.Context

	rock := model.Tags{model.TagGenre: []string{"Rock"}}

	BeforeEach(func() {
		ctx = GinkgoT().Context()
		ds = &getRandomSpy{DataStore: persistence.New(db.Db())}
		agent = &localAgent{ds: ds}
	})

	It("excludes the seed track from its own similars", func() {
		putSongs(ctx, ds,
			model.MediaFile{ID: "seed-1", Title: "Seed", Tags: rock},
			model.MediaFile{ID: "rel-1", Title: "Related", Tags: rock},
		)

		songs, err := agent.GetSimilarSongsByTrack(ctx, "seed-1", "Seed", "", "", 10)

		Expect(err).ToNot(HaveOccurred())
		names := slice.Map(songs, func(s Song) string { return s.Name })
		Expect(names).To(ConsistOf("Related"))
	})

	It("queries the indexed genre join for the seed's own genres, skipping missing files", func() {
		putSongs(ctx, ds,
			model.MediaFile{ID: "seed-4", Title: "Seed", Tags: rock},
			model.MediaFile{ID: "rock", Title: "Rock", Tags: rock},
			model.MediaFile{ID: "jazz", Title: "Jazz", Tags: model.Tags{model.TagGenre: []string{"Jazz"}}},
			model.MediaFile{ID: "gone", Title: "Missing Rock", Tags: rock, Missing: true},
		)

		songs, err := agent.GetSimilarSongsByTrack(ctx, "seed-4", "Seed", "", "", 10)
		Expect(err).ToNot(HaveOccurred())
		Expect(songs).To(ConsistOf(Song{ID: "rock", Name: "Rock"}))

		Expect(ds.options).To(HaveLen(1))
		sql, _, sqlErr := ds.options[0].Filters.ToSql()
		Expect(sqlErr).ToNot(HaveOccurred())
		Expect(sql).To(ContainSubstring("media_file_tags"), "must use the indexed join, not a json_tree scan")
	})

	It("returns the library id so the matcher can resolve the song", func() {
		// Without the id the matcher falls through to its MBID/title phases and resolves nothing,
		// so the local fallback silently returns an empty mix.
		putSongs(ctx, ds,
			model.MediaFile{ID: "seed-3", Title: "Seed", Tags: rock},
			model.MediaFile{ID: "rel-3", Title: "Related", Tags: rock},
		)

		songs, err := agent.GetSimilarSongsByTrack(ctx, "seed-3", "Seed", "", "", 10)

		Expect(err).ToNot(HaveOccurred())
		Expect(songs).To(ContainElement(Song{ID: "rel-3", Name: "Related"}))
	})

	It("asks for one extra candidate so dropping the seed still fills the count", func() {
		putSongs(ctx, ds,
			model.MediaFile{ID: "a-seed", Title: "Seed", Tags: rock},
			model.MediaFile{ID: "b-rel", Title: "Related", Tags: rock},
		)

		// Candidates come back in random order, so repeat until the seed has surely been drawn first.
		for range 20 {
			songs, err := agent.GetSimilarSongsByTrack(ctx, "a-seed", "Seed", "", "", 1)

			Expect(err).ToNot(HaveOccurred())
			Expect(songs).To(HaveLen(1))
			Expect(songs[0].Name).To(Equal("Related"))
		}
	})

	It("returns nil when the seed track has no genres", func() {
		putSongs(ctx, ds,
			model.MediaFile{ID: "seed-2", Title: "NoGenre"},
			model.MediaFile{ID: "other", Title: "Other", Tags: rock},
		)

		songs, err := agent.GetSimilarSongsByTrack(ctx, "seed-2", "NoGenre", "", "", 10)

		Expect(err).ToNot(HaveOccurred())
		Expect(songs).To(BeEmpty())
		// Without the early return an empty tag filter would scan the whole library.
		Expect(ds.options).To(BeEmpty(), "must not query at all")
	})
})

// getRandomSpy records the options of every MediaFile().GetRandom call, which still reaches the real repository.
type getRandomSpy struct {
	model.DataStore
	options []model.QueryOptions
}

func (s *getRandomSpy) MediaFile() model.MediaFileRepository {
	return getRandomSpyRepo{MediaFileRepository: s.DataStore.MediaFile(), spy: s}
}

type getRandomSpyRepo struct {
	model.MediaFileRepository
	spy *getRandomSpy
}

func (r getRandomSpyRepo) GetRandom(ctx context.Context, options ...model.QueryOptions) (model.MediaFiles, error) {
	r.spy.options = append(r.spy.options, options...)
	return r.MediaFileRepository.GetRandom(ctx, options...)
}

// putSongs stores the songs in library 1, along with the tags they carry.
func putSongs(ctx context.Context, ds model.DataStore, songs ...model.MediaFile) {
	GinkgoHelper()
	for _, mf := range songs {
		if tags := mf.Tags.FlattenAll(); len(tags) > 0 {
			Expect(ds.Tag().Add(ctx, model.DefaultLibraryID, tags...)).To(Succeed())
		}
	}
	persistencetest.PutMediaFiles(ctx, ds, songs...)
}
