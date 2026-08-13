package agents

import (
	"context"

	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("localAgent GetSimilarSongsByTrack", func() {
	var ds *tests.MockDataStore
	var mfRepo *tests.MockMediaFileRepo
	var agent *localAgent
	var ctx context.Context

	BeforeEach(func() {
		ctx = context.Background()
		mfRepo = &tests.MockMediaFileRepo{}
		ds = &tests.MockDataStore{MockedMediaFile: mfRepo}
		agent = &localAgent{ds: ds}
	})

	It("returns songs sharing the seed's genres, excluding the seed", func() {
		seed := model.MediaFile{ID: "seed-1", Title: "Seed", Tags: model.Tags{model.TagGenre: []string{"Rock"}}}
		related := model.MediaFile{ID: "rel-1", Title: "Related", Tags: model.Tags{model.TagGenre: []string{"Rock"}}}
		// SetData keys by ID; a duplicate "seed-1" entry would clobber the real seed.
		mfRepo.SetData(model.MediaFiles{seed, related})

		songs, err := agent.GetSimilarSongsByTrack(ctx, "seed-1", "Seed", "", "", 10)

		Expect(err).ToNot(HaveOccurred())
		ids := make([]string, 0, len(songs))
		for _, s := range songs {
			ids = append(ids, s.Name)
		}
		Expect(ids).To(ContainElement("Related"))
		Expect(ids).ToNot(ContainElement("Seed"))
	})

	// The mock ignores QueryOptions.Filters, so assert the predicate itself: otherwise this spec
	// would pass just as well with no genre filter at all.
	It("queries the indexed genre join for the seed's own genres, skipping missing files", func() {
		rock := model.NewTag(model.TagGenre, "Rock")
		seed := model.MediaFile{ID: "seed-4", Title: "Seed", Tags: model.Tags{model.TagGenre: []string{"Rock"}}}
		mfRepo.SetData(model.MediaFiles{seed})

		_, err := agent.GetSimilarSongsByTrack(ctx, "seed-4", "Seed", "", "", 10)
		Expect(err).ToNot(HaveOccurred())

		sql, args, sqlErr := mfRepo.Options.Filters.ToSql()
		Expect(sqlErr).ToNot(HaveOccurred())
		Expect(sql).To(ContainSubstring("media_file_tags"), "must use the indexed join, not a json_tree scan")
		Expect(sql).To(ContainSubstring("missing"))
		Expect(args).To(ContainElement(rock.ID), "must filter on the seed's own genre tag id")
		Expect(args).ToNot(ContainElement(model.NewTag(model.TagGenre, "Jazz").ID))
	})

	It("returns the library id so the matcher can resolve the song", func() {
		seed := model.MediaFile{ID: "seed-3", Title: "Seed", Tags: model.Tags{model.TagGenre: []string{"Rock"}}}
		// Without the id the matcher falls through to its MBID/title phases and resolves nothing,
		// so the local fallback silently returns an empty mix.
		related := model.MediaFile{ID: "rel-3", Title: "Related", Tags: model.Tags{model.TagGenre: []string{"Rock"}}}
		mfRepo.SetData(model.MediaFiles{seed, related})

		songs, err := agent.GetSimilarSongsByTrack(ctx, "seed-3", "Seed", "", "", 10)

		Expect(err).ToNot(HaveOccurred())
		Expect(songs).To(ContainElement(Song{ID: "rel-3", Name: "Related"}))
	})

	It("returns nil when the seed track has no genres", func() {
		seed := model.MediaFile{ID: "seed-2", Title: "NoGenre"}
		mfRepo.SetData(model.MediaFiles{seed})

		songs, err := agent.GetSimilarSongsByTrack(ctx, "seed-2", "NoGenre", "", "", 10)

		Expect(err).ToNot(HaveOccurred())
		Expect(songs).To(BeEmpty())
	})
})
