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
		persistence.Exists("json_tree(Participants, '$.albumartist')", Eq{"value": artistId}),
	}
	if conf.Server.SubsonicArtistParticipations {
		filters = append(filters,
			persistence.Exists("json_tree(Participants, '$.artist')", Eq{"value": artistId}),
		)
	}
	return addDefaultFilters(Options{
		Sort:    "max_year",
		Filters: Or(filters),
	})
}

func AlbumsByYear(fromYear, toYear int) Options {
	sortOption := "max_year, name"
	if fromYear > toYear {
		fromYear, toYear = toYear, fromYear
		sortOption = "max_year desc, name"
	}
	return addDefaultFilters(Options{
		Sort: sortOption,
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
		ff = append(ff, Eq{"genre.name": genre})
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

func SongWithLyrics(artist, title string) Options {
	return addDefaultFilters(Options{
		Sort:    "updated_at",
		Order:   "desc",
		Max:     1,
		Filters: And{Eq{"artist": artist, "title": title}, NotEq{"lyrics": ""}},
	})
}

func ByGenre(genre string) Options {
	return addDefaultFilters(Options{
		Sort: "name asc",
		Filters: persistence.Exists("json_tree(tags)", And{
			Like{"value": genre},
			NotEq{"atom": nil},
		}),
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
