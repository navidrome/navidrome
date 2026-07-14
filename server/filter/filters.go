package filter

import (
	"time"

	. "github.com/Masterminds/squirrel"
	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/persistence"
)

type Options = model.QueryOptions

var defaultFilters = Eq{"missing": false}

func addDefaultFilters(options Options) Options {
	if options.Filters == nil {
		options.Filters = defaultFilters
	} else {
		options.Filters = And{defaultFilters, options.Filters}
	}
	return options
}

func AlbumsByNewest() Options {
	return addDefaultFilters(addDefaultFilters(Options{Sort: "recently_added", Order: "desc"}))
}

func AlbumsByRecent() Options {
	return addDefaultFilters(Options{Sort: "playDate", Order: "desc", Filters: Gt{"play_date": time.Time{}}})
}

func AlbumsByFrequent() Options {
	return addDefaultFilters(Options{Sort: "playCount", Order: "desc", Filters: Gt{"play_count": 0}})
}

func AlbumsByRandom() Options {
	return addDefaultFilters(Options{Sort: "random"})
}

func AlbumsByName() Options {
	return addDefaultFilters(Options{Sort: "name"})
}

func AlbumsByArtist() Options {
	return addDefaultFilters(Options{Sort: "artist"})
}

func AlbumsByArtistID(artistId string) Options {
	filters := []Sqlizer{
		persistence.Exists("json_tree(participants, '$.albumartist')", Eq{"value": artistId}),
	}
	if conf.Server.Subsonic.ArtistParticipations {
		filters = append(filters,
			persistence.Exists("json_tree(participants, '$.artist')", Eq{"value": artistId}),
		)
	}
	return addDefaultFilters(Options{
		Sort:    "max_year",
		Filters: Or(filters),
	})
}

// AlbumsByContributingArtistID matches albums where the artist performs on a track but is not the
// album artist — Jellyfin's "Featured On". The disjoint complement of AlbumsByArtistID, so an
// artist's own discography never leaks into it.
func AlbumsByContributingArtistID(artistId string) Options {
	return addDefaultFilters(Options{
		Sort: "max_year",
		Filters: And{
			persistence.Exists("json_tree(participants, '$.artist')", Eq{"value": artistId}),
			persistence.NotExists("json_tree(participants, '$.albumartist')", Eq{"value": artistId}),
		},
	})
}

func AlbumsByYear(fromYear, toYear int) Options {
	orderOption := ""
	if fromYear > toYear {
		fromYear, toYear = toYear, fromYear
		orderOption = "desc"
	}
	return addDefaultFilters(Options{
		Sort:  "max_year",
		Order: orderOption,
		Filters: Or{
			And{
				GtOrEq{"min_year": fromYear},
				LtOrEq{"min_year": toYear},
			},
			And{
				GtOrEq{"max_year": fromYear},
				LtOrEq{"max_year": toYear},
			},
		},
	})
}

func SongsByAlbum(albumId string) Options {
	return addDefaultFilters(Options{
		Filters: Eq{"album_id": albumId},
		Sort:    "album",
	})
}

// SongsByArtistID matches media files where the artist participates as album or track artist, in
// album order. Semi-joins media_file_artists; scanning the participants JSON is ~10x slower at scale.
func SongsByArtistID(artistId string) Options {
	return addDefaultFilters(Options{
		Sort: "album",
		Filters: Expr(
			"media_file.id IN (SELECT media_file_id FROM media_file_artists WHERE artist_id = ? AND role IN (?, ?))",
			artistId, model.RoleArtist.String(), model.RoleAlbumArtist.String()),
	})
}

func SongsByGenreAndYearRange(genre string, fromYear, toYear int) Options {
	options := Options{}
	ff := And{}
	if genre != "" {
		ff = append(ff, filterByGenre(genre))
	}
	if fromYear != 0 {
		ff = append(ff, GtOrEq{"year": fromYear})
	}
	if toYear != 0 {
		ff = append(ff, LtOrEq{"year": toYear})
	}
	options.Filters = ff
	return addDefaultFilters(options)
}

func ApplyLibraryFilter(opts Options, musicFolderIds []int) Options {
	if len(musicFolderIds) == 0 {
		return opts
	}

	libraryFilter := Eq{"library_id": musicFolderIds}
	if opts.Filters == nil {
		opts.Filters = libraryFilter
	} else {
		opts.Filters = And{opts.Filters, libraryFilter}
	}

	return opts
}

// ApplyArtistLibraryFilter applies a filter to the given Options to ensure that only artists
// that are associated with the specified music folders are included in the results.
func ApplyArtistLibraryFilter(opts Options, musicFolderIds []int) Options {
	if len(musicFolderIds) == 0 {
		return opts
	}

	artistLibraryFilter := Eq{"library_artist.library_id": musicFolderIds}
	if opts.Filters == nil {
		opts.Filters = artistLibraryFilter
	} else {
		opts.Filters = And{opts.Filters, artistLibraryFilter}
	}

	return opts
}

// ArtistsByRole restricts an artist query to artists appearing in the given role (album artist,
// performer, composer, ...) via library_artist.stats. An unknown role is ignored (no filter).
func ArtistsByRole(opts Options, role model.Role) Options {
	if _, ok := model.AllRoles[role.String()]; !ok {
		return opts
	}
	roleFilter := Expr("JSON_EXTRACT(library_artist.stats, '$." + role.String() + ".m') IS NOT NULL")
	if opts.Filters == nil {
		opts.Filters = roleFilter
	} else {
		opts.Filters = And{opts.Filters, roleFilter}
	}
	return opts
}

func ByGenre(genre string) Options {
	return addDefaultFilters(Options{
		Sort:    "name",
		Filters: filterByGenre(genre),
	})
}

// ByGenreID matches items (albums or songs) tagged with any of the given genre tag ids.
func ByGenreID(genreIds []string) Sqlizer {
	return genreTagFilter(Eq{"value": genreIds})
}

// ArtistsByGenreID matches artists credited as album artist on an album with any of the given
// genre tag ids. Non-correlated semi-join: the correlated EXISTS form rescans albums per artist row.
func ArtistsByGenreID(genreIds []string) Sqlizer {
	return Expr(
		`artist.id IN (SELECT jt.value FROM album, json_tree(album.participants, '$.albumartist') jt
			WHERE jt.atom IS NOT NULL AND ?)`,
		genreTagFilter(Eq{"value": genreIds}),
	)
}

// genreTagFilter builds an EXISTS over the genre entries in the tags JSON, matching each entry
// against cond (its name via Like, or its tag id via Eq/IN). Shared by the name- and id-based lookups.
func genreTagFilter(cond Sqlizer) Sqlizer {
	return persistence.Exists(`json_tree(tags, "$.genre")`, And{NotEq{"atom": nil}, cond})
}

func filterByGenre(genre string) Sqlizer {
	return genreTagFilter(Like{"value": genre})
}

func ByRating() Options {
	return addDefaultFilters(Options{Sort: "rating", Order: "desc", Filters: Gt{"rating": 0}})
}

func ByStarred() Options {
	return addDefaultFilters(Options{Sort: "starred_at", Order: "desc", Filters: Eq{"starred": true}})
}

func ArtistsByStarred() Options {
	return Options{Sort: "starred_at", Order: "desc", Filters: Eq{"starred": true}}
}
