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
	roles := []model.Role{model.RoleAlbumArtist}
	if conf.Server.Subsonic.ArtistParticipations {
		roles = append(roles, model.RoleArtist)
	}
	return addDefaultFilters(Options{
		Sort:    "max_year",
		Filters: persistence.ParticipantIDFilter("album", artistId, roles...),
	})
}

// AlbumsByContributingArtistID matches albums where the artist performs on a track but is not the
// album artist — Jellyfin's "Featured On". The disjoint complement of AlbumsByArtistID, so an
// artist's own discography never leaks into it.
func AlbumsByContributingArtistID(artistId string) Options {
	return addDefaultFilters(Options{
		Sort: "max_year",
		Filters: And{
			persistence.ParticipantIDFilter("album", artistId, model.RoleArtist),
			persistence.NotParticipantIDFilter("album", artistId, model.RoleAlbumArtist),
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
// album order.
func SongsByArtistID(artistId string) Options {
	return addDefaultFilters(Options{
		Sort:    "album",
		Filters: persistence.ParticipantIDFilter("media_file", artistId, model.RoleArtist, model.RoleAlbumArtist),
	})
}

func SongsByGenreAndYearRange(genre string, fromYear, toYear int) Options {
	options := Options{}
	ff := And{}
	if genre != "" {
		ff = append(ff, persistence.SongGenres.ByName(genre))
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
	roleFilter := Expr("library_artist.stats->'" + role.String() + "'->>'m' IS NOT NULL")
	if opts.Filters == nil {
		opts.Filters = roleFilter
	} else {
		opts.Filters = And{opts.Filters, roleFilter}
	}
	return opts
}

// SongsByGenreID / AlbumsByGenreID (by tag id) and AlbumsByGenre / SongsByGenre (by name, wrapped
// as Options for Subsonic) delegate to the persistence genre filters, which own the join schema.
func SongsByGenreID(genreIds []string) Sqlizer  { return persistence.SongGenres.ByID(genreIds) }
func AlbumsByGenreID(genreIds []string) Sqlizer { return persistence.AlbumGenres.ByID(genreIds) }

func AlbumsByGenre(genre string) Options {
	return addDefaultFilters(Options{Sort: "name", Filters: persistence.AlbumGenres.ByName(genre)})
}

func SongsByGenre(genre string) Options {
	return addDefaultFilters(Options{Sort: "name", Filters: persistence.SongGenres.ByName(genre)})
}

// ByAlbumID matches media files belonging to any of the given albums.
func ByAlbumID(albumIds []string) Sqlizer {
	return Eq{"album_id": albumIds}
}

// AlbumsByYears matches albums whose production year (max_year) is in years.
func AlbumsByYears(years []int) Sqlizer {
	return Eq{"max_year": years}
}

// SongsByYears matches media files whose year is in years.
func SongsByYears(years []int) Sqlizer {
	return Eq{"year": years}
}

func ArtistsByGenreID(genreIds []string) Sqlizer {
	return persistence.AlbumArtistsByGenreID(genreIds)
}

// tagIDFilter builds an EXISTS over the given tag role's entries in the tags JSON, matching each
// entry against cond (its name via Like, or its tag id via Eq/IN).
func tagIDFilter(tagName string, cond Sqlizer) Sqlizer {
	return persistence.Exists(`json_tree(tags, "$.`+tagName+`")`, And{NotEq{"atom": nil}, cond})
}

// ByStudioID matches items (albums or songs) whose record-label tag id is in ids.
func ByStudioID(ids []string) Sqlizer {
	return tagIDFilter("recordlabel", Eq{"value": ids})
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
