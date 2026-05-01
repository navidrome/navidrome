package persistence

import (
	"time"

	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/criteria"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Smart playlist criteria SQL", func() {
	BeforeEach(func() {
		criteria.AddRoles([]string{"artist", "composer", "producer"})
		criteria.AddTagNames([]string{"genre", "mood", "releasetype", "recordingdate"})
		criteria.AddNumericTags([]string{"rate"})
	})

	DescribeTable("expressions",
		func(expr criteria.Expression, expectedSQL string, expectedArgs ...any) {
			sqlizer, err := newSmartPlaylistCriteria(criteria.Criteria{Expression: expr}).Where()
			Expect(err).ToNot(HaveOccurred())

			sql, args, err := sqlizer.ToSql()
			Expect(err).ToNot(HaveOccurred())
			Expect(sql).To(Equal(expectedSQL))
			Expect(args).To(HaveExactElements(expectedArgs...))
		},
		Entry("all group",
			criteria.All{criteria.Contains{"title": "love"}, criteria.Gt{"rating": 3}},
			"(media_file.title LIKE ? AND COALESCE(annotation.rating, 0) > ?)", "%love%", 3),
		Entry("any group",
			criteria.Any{criteria.Is{"title": "Low Rider"}, criteria.Is{"album": "Best Of"}},
			"(media_file.title = ? OR media_file.album = ?)", "Low Rider", "Best Of"),
		Entry("is string", criteria.Is{"title": "Low Rider"}, "media_file.title = ?", "Low Rider"),
		Entry("is bool", criteria.Is{"loved": true}, "COALESCE(annotation.starred, false) = ?", true),
		Entry("is numeric list", criteria.Is{"library_id": []int{1, 2}}, "media_file.library_id IN (?,?)", 1, 2),
		Entry("is not", criteria.IsNot{"title": "Low Rider"}, "media_file.title <> ?", "Low Rider"),
		Entry("gt", criteria.Gt{"playCount": 10}, "COALESCE(annotation.play_count, 0) > ?", 10),
		Entry("lt", criteria.Lt{"playCount": 10}, "COALESCE(annotation.play_count, 0) < ?", 10),
		Entry("contains", criteria.Contains{"title": "Low Rider"}, "media_file.title LIKE ?", "%Low Rider%"),
		Entry("not contains", criteria.NotContains{"title": "Low Rider"}, "media_file.title NOT LIKE ?", "%Low Rider%"),
		Entry("starts with", criteria.StartsWith{"title": "Low Rider"}, "media_file.title LIKE ?", "Low Rider%"),
		Entry("ends with", criteria.EndsWith{"title": "Low Rider"}, "media_file.title LIKE ?", "%Low Rider"),
		Entry("in range", criteria.InTheRange{"year": []int{1980, 1990}}, "(media_file.year >= ? AND media_file.year <= ?)", 1980, 1990),
		Entry("before", criteria.Before{"lastPlayed": time.Date(2021, 10, 1, 0, 0, 0, 0, time.Local)}, "annotation.play_date < ?", time.Date(2021, 10, 1, 0, 0, 0, 0, time.Local)),
		Entry("after", criteria.After{"lastPlayed": time.Date(2021, 10, 1, 0, 0, 0, 0, time.Local)}, "annotation.play_date > ?", time.Date(2021, 10, 1, 0, 0, 0, 0, time.Local)),
		Entry("in playlist", criteria.InPlaylist{"id": "deadbeef-dead-beef"}, "media_file.id IN (SELECT media_file_id FROM playlist_tracks pl LEFT JOIN playlist on pl.playlist_id = playlist.id WHERE (pl.playlist_id = ? AND playlist.public = ?))", "deadbeef-dead-beef", 1),
		Entry("not in playlist", criteria.NotInPlaylist{"id": "deadbeef-dead-beef"}, "media_file.id NOT IN (SELECT media_file_id FROM playlist_tracks pl LEFT JOIN playlist on pl.playlist_id = playlist.id WHERE (pl.playlist_id = ? AND playlist.public = ?))", "deadbeef-dead-beef", 1),
		Entry("album annotation", criteria.Gt{"albumRating": 3}, "COALESCE(album_annotation.rating, 0) > ?", 3),
		Entry("artist annotation", criteria.Is{"artistLoved": true}, "COALESCE(artist_annotation.starred, false) = ?", true),
		Entry("tag is", criteria.Is{"genre": "Rock"}, "exists (select 1 from json_tree(media_file.tags, '$.genre') where key='value' and value = ?)", "Rock"),
		Entry("tag is not", criteria.IsNot{"genre": "Rock"}, "not exists (select 1 from json_tree(media_file.tags, '$.genre') where key='value' and value = ?)", "Rock"),
		Entry("tag contains", criteria.Contains{"genre": "Rock"}, "exists (select 1 from json_tree(media_file.tags, '$.genre') where key='value' and value LIKE ?)", "%Rock%"),
		Entry("tag not contains", criteria.NotContains{"genre": "Rock"}, "not exists (select 1 from json_tree(media_file.tags, '$.genre') where key='value' and value LIKE ?)", "%Rock%"),
		Entry("numeric tag", criteria.Lt{"rate": 6}, "exists (select 1 from json_tree(media_file.tags, '$.rate') where key='value' and CAST(value AS REAL) < ?)", 6),
		Entry("tag alias", criteria.Is{"albumtype": "album"}, "exists (select 1 from json_tree(media_file.tags, '$.releasetype') where key='value' and value = ?)", "album"),
		Entry("field alias via tag registration", criteria.Is{"recordingdate": "2024-01-01"}, "media_file.date = ?", "2024-01-01"),
		Entry("role is", criteria.Is{"artist": "u2"}, "exists (select 1 from json_tree(media_file.participants, '$.artist') where key='name' and value = ?)", "u2"),
		Entry("role contains", criteria.Contains{"composer": "Lennon"}, "exists (select 1 from json_tree(media_file.participants, '$.composer') where key='name' and value LIKE ?)", "%Lennon%"),
		Entry("role not contains", criteria.NotContains{"artist": "u2"}, "not exists (select 1 from json_tree(media_file.participants, '$.artist') where key='name' and value LIKE ?)", "%u2%"),
		// ReplayGain fields
		Entry("rgAlbumGain is", criteria.Is{"rgAlbumGain": 0}, "media_file.rg_album_gain = ?", 0),
		Entry("rgAlbumGain gt", criteria.Gt{"rgAlbumGain": -6.0}, "media_file.rg_album_gain > ?", -6.0),
		Entry("rgTrackPeak lt", criteria.Lt{"rgTrackPeak": 1.0}, "media_file.rg_track_peak < ?", 1.0),
		// isMissing — tags
		Entry("isMissing tag [true]", criteria.IsMissing{"genre": true},
			"not exists (select 1 from json_tree(media_file.tags, '$.genre') where key='value')"),
		Entry("isMissing tag [false]", criteria.IsMissing{"genre": false},
			"exists (select 1 from json_tree(media_file.tags, '$.genre') where key='value')"),
		// isMissing — roles
		Entry("isMissing role [true]", criteria.IsMissing{"artist": true},
			"not exists (select 1 from json_tree(media_file.participants, '$.artist') where key='name')"),
		Entry("isMissing role [false]", criteria.IsMissing{"artist": false},
			"exists (select 1 from json_tree(media_file.participants, '$.artist') where key='name')"),
		// isPresent — tags
		Entry("isPresent tag [true]", criteria.IsPresent{"genre": true},
			"exists (select 1 from json_tree(media_file.tags, '$.genre') where key='value')"),
		Entry("isPresent tag [false]", criteria.IsPresent{"genre": false},
			"not exists (select 1 from json_tree(media_file.tags, '$.genre') where key='value')"),
		// isPresent — roles
		Entry("isPresent role [true]", criteria.IsPresent{"composer": true},
			"exists (select 1 from json_tree(media_file.participants, '$.composer') where key='name')"),
		Entry("isPresent role [false]", criteria.IsPresent{"composer": false},
			"not exists (select 1 from json_tree(media_file.participants, '$.composer') where key='name')"),
	)

	Describe("playlist permissions", func() {
		It("allows public or same-owner playlist references for regular users", func() {
			sqlizer, err := newSmartPlaylistCriteria(
				criteria.Criteria{Expression: criteria.InPlaylist{"id": "deadbeef-dead-beef"}},
				withSmartPlaylistOwner(model.User{ID: "owner-id", IsAdmin: false}),
			).Where()
			Expect(err).ToNot(HaveOccurred())

			sql, args, err := sqlizer.ToSql()
			Expect(err).ToNot(HaveOccurred())
			Expect(sql).To(Equal("media_file.id IN (SELECT media_file_id FROM playlist_tracks pl LEFT JOIN playlist on pl.playlist_id = playlist.id WHERE (pl.playlist_id = ? AND (playlist.public = ? OR playlist.owner_id = ?)))"))
			Expect(args).To(HaveExactElements("deadbeef-dead-beef", 1, "owner-id"))
		})

		It("allows all playlist references for admins", func() {
			sqlizer, err := newSmartPlaylistCriteria(
				criteria.Criteria{Expression: criteria.InPlaylist{"id": "deadbeef-dead-beef"}},
				withSmartPlaylistOwner(model.User{ID: "admin-id", IsAdmin: true}),
			).Where()
			Expect(err).ToNot(HaveOccurred())

			sql, args, err := sqlizer.ToSql()
			Expect(err).ToNot(HaveOccurred())
			Expect(sql).To(Equal("media_file.id IN (SELECT media_file_id FROM playlist_tracks pl LEFT JOIN playlist on pl.playlist_id = playlist.id WHERE (pl.playlist_id = ?))"))
			Expect(args).To(HaveExactElements("deadbeef-dead-beef"))
		})
	})

	It("builds relative date expressions", func() {
		sqlizer, err := newSmartPlaylistCriteria(criteria.Criteria{Expression: criteria.InTheLast{"lastPlayed": 30}}).Where()
		Expect(err).ToNot(HaveOccurred())

		sql, args, err := sqlizer.ToSql()
		Expect(err).ToNot(HaveOccurred())
		Expect(sql).To(Equal("annotation.play_date > ?"))
		Expect(args).To(HaveExactElements(startOfPeriod(30, time.Now())))
	})

	It("builds negated relative date expressions", func() {
		sqlizer, err := newSmartPlaylistCriteria(criteria.Criteria{Expression: criteria.NotInTheLast{"lastPlayed": 30}}).Where()
		Expect(err).ToNot(HaveOccurred())

		sql, args, err := sqlizer.ToSql()
		Expect(err).ToNot(HaveOccurred())
		Expect(sql).To(Equal("(annotation.play_date < ? OR annotation.play_date IS NULL)"))
		Expect(args).To(HaveExactElements(startOfPeriod(30, time.Now())))
	})

	It("returns an error for unknown fields", func() {
		_, err := newSmartPlaylistCriteria(criteria.Criteria{Expression: criteria.EndsWith{"unknown": "value"}}).Where()

		Expect(err).To(MatchError("invalid field in criteria: unknown"))
	})

	It("returns an error when isMissing is used with a regular field", func() {
		_, err := newSmartPlaylistCriteria(criteria.Criteria{Expression: criteria.IsMissing{"year": true}}).Where()
		Expect(err).To(MatchError(ContainSubstring("isMissing/isPresent operator is only supported for tag and role fields")))
	})

	It("returns an error when isPresent is used with a regular field", func() {
		_, err := newSmartPlaylistCriteria(criteria.Criteria{Expression: criteria.IsPresent{"title": true}}).Where()
		Expect(err).To(MatchError(ContainSubstring("isMissing/isPresent operator is only supported for tag and role fields")))
	})

	It("returns an error when isMissing has a non-boolean value", func() {
		_, err := newSmartPlaylistCriteria(criteria.Criteria{Expression: criteria.IsMissing{"genre": "hello"}}).Where()
		Expect(err).To(MatchError(ContainSubstring("invalid boolean value for 'missing' expression")))
	})

	Describe("sort", func() {
		It("sorts by regular fields", func() {
			Expect(newSmartPlaylistCriteria(criteria.Criteria{Sort: "title"}).OrderBy()).To(Equal("media_file.title asc"))
		})

		It("sorts by tag fields", func() {
			Expect(newSmartPlaylistCriteria(criteria.Criteria{Sort: "genre"}).OrderBy()).To(Equal("COALESCE(json_extract(media_file.tags, '$.genre[0].value'), '') asc"))
		})

		It("sorts by role fields", func() {
			Expect(newSmartPlaylistCriteria(criteria.Criteria{Sort: "artist"}).OrderBy()).To(Equal("COALESCE(json_extract(media_file.participants, '$.artist[0].name'), '') asc"))
		})

		It("casts numeric tags when sorting", func() {
			Expect(newSmartPlaylistCriteria(criteria.Criteria{Sort: "rate"}).OrderBy()).To(Equal("CAST(COALESCE(json_extract(media_file.tags, '$.rate[0].value'), '') AS REAL) asc"))
		})

		It("sorts by albumtype alias", func() {
			Expect(newSmartPlaylistCriteria(criteria.Criteria{Sort: "albumtype"}).OrderBy()).To(Equal("COALESCE(json_extract(media_file.tags, '$.releasetype[0].value'), '') asc"))
		})

		It("sorts by random", func() {
			Expect(newSmartPlaylistCriteria(criteria.Criteria{Sort: "random"}).OrderBy()).To(Equal("random() asc"))
		})

		It("sorts by multiple fields", func() {
			Expect(newSmartPlaylistCriteria(criteria.Criteria{Sort: "title,-rating"}).OrderBy()).To(Equal("media_file.title asc, COALESCE(annotation.rating, 0) desc"))
		})

		It("reverts order when order is desc", func() {
			Expect(newSmartPlaylistCriteria(criteria.Criteria{Sort: "-date,artist", Order: "desc"}).OrderBy()).To(Equal("media_file.date asc, COALESCE(json_extract(media_file.participants, '$.artist[0].name'), '') desc"))
		})

		It("ignores invalid sort fields", func() {
			Expect(newSmartPlaylistCriteria(criteria.Criteria{Sort: "bogus,title"}).OrderBy()).To(Equal("media_file.title asc"))
		})
	})

	It("has SQL mappings for all non-tag/non-role criteria fields", func() {
		for _, name := range criteria.AllFieldNames() {
			info, ok := criteria.LookupField(name)
			Expect(ok).To(BeTrue(), "field %q registered but LookupField fails", name)
			if info.IsTag || info.IsRole {
				continue
			}
			_, hasSQLField := smartPlaylistFields[info.Name()]
			Expect(hasSQLField).To(BeTrue(), "criteria field %q (name=%q) has no entry in smartPlaylistFields", name, info.Name())
		}
	})

	Describe("joins", func() {
		It("excludes sort-only joins from expression joins", func() {
			c := criteria.Criteria{Expression: criteria.All{criteria.Contains{"title": "love"}}, Sort: "albumRating"}
			cSQL := newSmartPlaylistCriteria(c)

			Expect(cSQL.ExpressionJoins()).To(Equal(smartPlaylistJoinNone))
			Expect(cSQL.RequiredJoins().has(smartPlaylistJoinAlbumAnnotation)).To(BeTrue())
		})

		It("includes expression-based joins", func() {
			c := criteria.Criteria{Expression: criteria.All{criteria.Gt{"albumRating": 3}}}

			Expect(newSmartPlaylistCriteria(c).ExpressionJoins().has(smartPlaylistJoinAlbumAnnotation)).To(BeTrue())
		})

		It("detects nested album and artist joins", func() {
			c := criteria.Criteria{Expression: criteria.All{
				criteria.Any{criteria.All{criteria.Is{"albumLoved": true}}},
				criteria.Any{criteria.Gt{"artistPlayCount": 10}},
			}}

			joins := newSmartPlaylistCriteria(c).RequiredJoins()
			Expect(joins.has(smartPlaylistJoinAlbumAnnotation)).To(BeTrue())
			Expect(joins.has(smartPlaylistJoinArtistAnnotation)).To(BeTrue())
		})

		It("detects join types from sort fields with direction prefixes", func() {
			c := criteria.Criteria{Expression: criteria.All{criteria.Contains{"title": "love"}}, Sort: "-artistRating"}

			Expect(newSmartPlaylistCriteria(c).RequiredJoins().has(smartPlaylistJoinArtistAnnotation)).To(BeTrue())
		})
	})
})
