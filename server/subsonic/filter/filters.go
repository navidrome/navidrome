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

func SongsByRandom(genre string, fromYear, toYear int) Options {
	options := Options{
		Sort: "random",
	}
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

func SongsByArtistTitleWithLyricsFirst(artist, title string) Options {
	return addDefaultFilters(Options{
		Sort:  "lyrics, updated_at",
		Order: "desc",
		Max:   1,
		Filters: And{
			Eq{"title": title},
			Or{
				persistence.Exists("json_tree(participants, '$.albumartist')", Eq{"value": artist}),
				persistence.Exists("json_tree(participants, '$.artist')", Eq{"value": artist}),
			},
		},
	})
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

func ByGenre(genre string) Options {
	return addDefaultFilters(Options{
		Sort:    "name",
		Filters: filterByGenre(genre),
	})
}

func filterByGenre(genre string) Sqlizer {
	return persistence.Exists(`json_tree(tags, "$.genre")`, And{
		Like{"value": genre},
		NotEq{"atom": nil},
	})
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
