package filter

import (
	"fmt"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/persistence"
)

type Options = model.QueryOptions

func AlbumsByNewest() Options {
	return Options{Sort: "recently_added", Order: "desc"}
}

func AlbumsByRecent() Options {
	return Options{Sort: "playDate", Order: "desc", Filters: squirrel.Gt{"play_date": time.Time{}}}
}

func AlbumsByFrequent() Options {
	return Options{Sort: "playCount", Order: "desc", Filters: squirrel.Gt{"play_count": 0}}
}

func AlbumsByRandom() Options {
	return Options{Sort: "random"}
}

func AlbumsByName() Options {
	return Options{Sort: "name"}
}

func AlbumsByArtist() Options {
	return Options{Sort: "artist"}
}

func AlbumsByArtistID(artistId string) Options {
	var filters squirrel.Sqlizer
	if conf.Server.SubsonicArtistParticipations {
		filters = squirrel.Like{"participants": fmt.Sprintf(`%%"%s"%%`, artistId)}
	} else {
		filters = squirrel.Eq{"album_artist_id": artistId}
	}
	return Options{
		Sort:    "max_year",
		Filters: filters,
	}
}

func AlbumsByYear(fromYear, toYear int) Options {
	sortOption := "max_year, name"
	if fromYear > toYear {
		fromYear, toYear = toYear, fromYear
		sortOption = "max_year desc, name"
	}
	return Options{
		Sort: sortOption,
		Filters: squirrel.Or{
			squirrel.And{
				squirrel.GtOrEq{"min_year": fromYear},
				squirrel.LtOrEq{"min_year": toYear},
			},
			squirrel.And{
				squirrel.GtOrEq{"max_year": fromYear},
				squirrel.LtOrEq{"max_year": toYear},
			},
		},
	}
}

func SongsByAlbum(albumId string) Options {
	return Options{
		Filters: squirrel.Eq{"album_id": albumId},
		Sort:    "album",
	}
}

func SongsByRandom(genre string, fromYear, toYear int) Options {
	options := Options{
		Sort: "random",
	}
	ff := squirrel.And{}
	if genre != "" {
		ff = append(ff, squirrel.Eq{"genre.name": genre})
	}
	if fromYear != 0 {
		ff = append(ff, squirrel.GtOrEq{"year": fromYear})
	}
	if toYear != 0 {
		ff = append(ff, squirrel.LtOrEq{"year": toYear})
	}
	options.Filters = ff
	return options
}

func SongsWithLyrics(artist, title string) Options {
	return Options{
		Sort:    "updated_at",
		Order:   "desc",
		Filters: squirrel.And{squirrel.Eq{"artist": artist, "title": title}, squirrel.NotEq{"lyrics": ""}},
	}
}

func ByGenre(genre string) Options {
	return Options{
		Sort: "name asc",
		Filters: persistence.Exists("json_tree(tags)", squirrel.And{
			squirrel.Like{"value": genre},
			squirrel.NotEq{"atom": nil},
		}),
	}
}

func ByRating() Options {
	return Options{Sort: "rating", Order: "desc", Filters: squirrel.Gt{"rating": 0}}
}

func ByStarred() Options {
	return Options{Sort: "starred_at", Order: "desc", Filters: squirrel.Eq{"starred": true}}
}
