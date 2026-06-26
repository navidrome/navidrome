package matcher

import (
	"github.com/navidrome/navidrome/core/agents"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("similarityRatio", func() {
	It("returns 1.0 for identical strings", func() {
		Expect(similarityRatio("hello", "hello")).To(BeNumerically("==", 1.0))
	})

	It("returns 0.0 for empty strings", func() {
		Expect(similarityRatio("", "test")).To(BeNumerically("==", 0.0))
		Expect(similarityRatio("test", "")).To(BeNumerically("==", 0.0))
	})

	It("returns high similarity for remastered suffix", func() {
		ratio := similarityRatio("paranoid android", "paranoid android remastered")
		Expect(ratio).To(BeNumerically(">=", 0.85))
	})

	It("returns high similarity for suffix additions like (Live)", func() {
		ratio := similarityRatio("bohemian rhapsody", "bohemian rhapsody live")
		Expect(ratio).To(BeNumerically(">=", 0.90))
	})

	It("returns high similarity for 'yesterday' variants (common prefix)", func() {
		ratio := similarityRatio("yesterday", "yesterday once more")
		Expect(ratio).To(BeNumerically(">=", 0.85))
	})

	It("returns low similarity for same suffix", func() {
		ratio := similarityRatio("postman (live)", "taxman (live)")
		Expect(ratio).To(BeNumerically("<", 0.85))
	})

	It("handles unicode characters", func() {
		ratio := similarityRatio("dont stop believin", "don't stop believin'")
		Expect(ratio).To(BeNumerically(">=", 0.85))
	})

	It("returns low similarity for completely different strings", func() {
		ratio := similarityRatio("abc", "xyz")
		Expect(ratio).To(BeNumerically("<", 0.5))
	})

	It("is symmetric", func() {
		ratio1 := similarityRatio("hello world", "hello")
		ratio2 := similarityRatio("hello", "hello world")
		Expect(ratio1).To(Equal(ratio2))
	})
})

var _ = Describe("matcher internals", func() {
	It("computeSpecificityLevel uses sanitizedTrack.artistMBIDs for artist-MBID levels", func() {
		q := songQuery{
			title:     "song",
			artists:   []queryArtist{{mbid: "artist-mbid-1"}},
			albumMBID: "album-mbid-1",
		}
		mf := model.MediaFile{Title: "Song", MbzAlbumID: "album-mbid-1"} // note: mf.MbzArtistID intentionally empty
		t := newSanitizedTrack(&mf, nil, map[string]struct{}{"artist-mbid-1": {}})
		Expect(computeSpecificityLevel(q, t, 0.85)).To(Equal(5))
	})

	It("computeSpecificityLevel maximizes the level over all query artists", func() {
		// First artist does not match; second matches by name with a matching album → level 3.
		q := songQuery{
			title: "song",
			album: "violator",
			artists: []queryArtist{
				{name: "no match"},
				{name: "depeche mode"},
			},
		}
		mf := model.MediaFile{Title: "Song", Artist: "Depeche Mode", Album: "Violator"}
		t := newSanitizedTrack(&mf, nil, nil)
		Expect(computeSpecificityLevel(q, t, 0.85)).To(Equal(3))
	})

	It("scores MBID specificity for any credited artist, not just the last", func() {
		q := songQuery{
			title: "song",
			artists: []queryArtist{
				{name: "drake", mbid: "mbz-drake"},
				{name: "future", mbid: "mbz-future"},
			},
			album: "wrong album", // force album mismatch so only MBID-level (2) is reachable, not 3+
		}
		// Track credits BOTH MBIDs; with the old last-wins string this would only match one.
		t := sanitizedTrack{
			mf:          &model.MediaFile{},
			title:       "song",
			artist:      "drake",
			album:       "some other album",
			artistMBIDs: map[string]struct{}{"mbz-drake": {}, "mbz-future": {}},
		}
		// Either artist's MBID matching yields level 2 (MBID, no album match). The point: it is
		// reached via mbz-future too, which the old code would have dropped.
		Expect(computeSpecificityLevel(q, t, 0.85)).To(Equal(2))
	})

	It("treats an ID-only artist (no name/MBID) as an identity match, unlocking album tiers", func() {
		// A plugin that supplies only a Navidrome artist ID: name and mbid are empty. The ID is the
		// strongest identity signal, so the track's album still elevates specificity above 0.
		q := songQuery{
			title:     "song",
			artists:   []queryArtist{{id: "artist-1"}},
			album:     "violator",
			albumMBID: "album-mbid-1",
		}
		// Track credits the owned artist ID; no MBID anywhere (untagged library / ID-only plugin).
		mf := model.MediaFile{Title: "Song", Album: "Violator", MbzAlbumID: "album-mbid-1"}
		t := newSanitizedTrack(&mf, map[string]struct{}{"artist-1": {}}, nil)
		// Album MBID matches → level 5 via the ID identity, where the old code scored 0.
		Expect(computeSpecificityLevel(q, t, 0.85)).To(Equal(5))

		// Same artist, album name matches but no album MBID → level 4 via the ID identity.
		q.albumMBID = ""
		mf2 := model.MediaFile{Title: "Song", Album: "Violator"}
		t2 := newSanitizedTrack(&mf2, map[string]struct{}{"artist-1": {}}, nil)
		Expect(computeSpecificityLevel(q, t2, 0.85)).To(Equal(4))
	})
})

var _ = Describe("groupQueries", func() {
	It("builds one query per unmatched song carrying all artists", func() {
		songs := []agents.Song{
			{Name: "Song A", Artists: []agents.Artist{{Name: "Drake"}, {Name: "Future"}}},
		}
		queries := groupQueries(songs, map[int]model.MediaFile{})
		Expect(queries).To(HaveLen(1))
		Expect(queries[0].index).To(Equal(0))
		Expect(queries[0].query.title).To(Equal("song a"))
		Expect(queries[0].query.artists).To(HaveLen(2))
		Expect(queries[0].query.artists[0].name).To(Equal("drake"))
		Expect(queries[0].query.artists[1].name).To(Equal("future"))
	})

	It("strips leading articles from artist names", func() {
		songs := []agents.Song{
			{Name: "Song A", Artists: []agents.Artist{{Name: "The Drake"}}},
		}
		queries := groupQueries(songs, map[int]model.MediaFile{})
		Expect(queries).To(HaveLen(1))
		Expect(queries[0].query.artists).To(HaveLen(1))
		Expect(queries[0].query.artists[0].name).To(Equal("drake"))
	})

	It("keeps an artist that carries only an ID (empty name)", func() {
		songs := []agents.Song{
			{Name: "Song A", Artists: []agents.Artist{{ID: "ar-x"}}},
		}
		queries := groupQueries(songs, map[int]model.MediaFile{})
		Expect(queries).To(HaveLen(1))
		Expect(queries[0].query.artists).To(HaveLen(1))
		Expect(queries[0].query.artists[0].id).To(Equal("ar-x"))
		Expect(queries[0].query.artists[0].name).To(Equal(""))
	})

	It("keeps an artist that carries only an MBID (empty id and name)", func() {
		// ListenBrainz collaborators arrive as MBID-only when the API supplies a combined display
		// name; the MBID is a usable identity signal and must not be dropped.
		songs := []agents.Song{
			{Name: "Song A", Artists: []agents.Artist{{MBID: "mbz-future"}}},
		}
		queries := groupQueries(songs, map[int]model.MediaFile{})
		Expect(queries).To(HaveLen(1))
		Expect(queries[0].query.artists).To(HaveLen(1))
		Expect(queries[0].query.artists[0].id).To(Equal(""))
		Expect(queries[0].query.artists[0].name).To(Equal(""))
		Expect(queries[0].query.artists[0].mbid).To(Equal("mbz-future"))
	})

	It("drops a fully-empty artist (no id, name, or mbid) but keeps usable ones", func() {
		songs := []agents.Song{
			{Name: "Song A", Artists: []agents.Artist{{}, {Name: "Future"}}},
		}
		queries := groupQueries(songs, map[int]model.MediaFile{})
		Expect(queries).To(HaveLen(1))
		Expect(queries[0].query.artists).To(HaveLen(1))
		Expect(queries[0].query.artists[0].name).To(Equal("future"))
	})

	It("skips already-matched songs and songs with no usable artist", func() {
		songs := []agents.Song{
			{Name: "Already Matched", Artists: []agents.Artist{{Name: "Drake"}}},
			{Name: "No Artist"},
			{Name: "Song C", Artists: []agents.Artist{{Name: "Future"}}},
		}
		queries := groupQueries(songs, map[int]model.MediaFile{0: {ID: "done"}})
		Expect(queries).To(HaveLen(1))
		Expect(queries[0].index).To(Equal(2))
		Expect(queries[0].query.artists[0].name).To(Equal("future"))
	})
})

var _ = Describe("bucketTracks", func() {
	It("scores tracks by how many of the query's artists they credit (overlap)", func() {
		r := resolvedArtists{
			byQuery: map[int]map[string]struct{}{
				0: {"ar-1": {}, "ar-2": {}},
			},
			mbid: map[string]string{},
		}
		trackA := model.MediaFile{ID: "a", Title: "A",
			Participants: artistParticipants(
				model.Artist{ID: "ar-1", OrderArtistName: "one"},
				model.Artist{ID: "ar-2", OrderArtistName: "two"},
			),
		}
		trackB := model.MediaFile{ID: "b", Title: "B",
			Participants: artistParticipants(model.Artist{ID: "ar-1", OrderArtistName: "one"}),
		}
		byQuery := r.bucketTracks(model.MediaFiles{trackA, trackB})
		Expect(byQuery[0]).To(HaveLen(2))
		overlaps := map[string]int{}
		for _, st := range byQuery[0] {
			overlaps[st.mf.ID] = st.overlap
		}
		Expect(overlaps["a"]).To(Equal(2))
		Expect(overlaps["b"]).To(Equal(1))
	})
})

var _ = Describe("resolveArtists ID fast-path", func() {
	It("owns an artist supplied by ID without a name match", func() {
		ctx := GinkgoT().Context()
		artistRepo := tests.CreateMockArtistRepo()
		artistRepo.SetData(model.Artists{
			{ID: "ar-x", Name: "Some Artist", OrderArtistName: "some artist", MbzArtistID: "mbz-x"},
		})
		ds := &tests.MockDataStore{MockedArtist: artistRepo}
		m := New(ds)

		queries := []indexedQuery{
			{index: 0, query: songQuery{title: "song", artists: []queryArtist{{id: "ar-x"}}}},
		}
		res, err := m.resolveArtists(ctx, queries)
		Expect(err).ToNot(HaveOccurred())
		Expect(res.byQuery[0]).To(HaveKey("ar-x"))
		Expect(res.allIDs).To(ContainElement("ar-x"))
		Expect(res.mbid["ar-x"]).To(Equal("mbz-x"))
	})
})

// artistParticipants builds a Participants map crediting the given artists under RoleArtist.
func artistParticipants(artists ...model.Artist) model.Participants {
	list := make(model.ParticipantList, len(artists))
	for i, a := range artists {
		list[i] = model.Participant{Artist: a}
	}
	return model.Participants{model.RoleArtist: list}
}
