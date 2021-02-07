package filter

import (
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/navidrome/navidrome/model"
)

type Options model.QueryOptions

func AlbumsByNewest() Options {
	return Options{Sort: "createdAt", Order: "desc"}
}

func AlbumsByRecent() Options {
	return Options{Sort: "playDate", Order: "desc", Filters: squirrel.Gt{"play_date": time.Time{}}}
}

func AlbumsByFrequent() Options {
	return Options{Sort: "playCount", Order: "desc", Filters: squirrel.Gt{"play_count": 0}}
}

func AlbumsByRandom() Options {
	return Options{Sort: "random()"}
}

func AlbumsByName() Options {
	return Options{Sort: "name"}
}

func AlbumsByArtist() Options {
	return Options{Sort: "artist"}
}

func AlbumsByStarred() Options {
	return Options{Sort: "starred_at", Order: "desc", Filters: squirrel.Eq{"starred": true}}
}

func AlbumsByRating() Options {
	return Options{Sort: "Rating", Order: "desc", Filters: squirrel.Gt{"rating": 0}}
}

func AlbumsByGenre(genre string) Options {
	return Options{
		Sort:    "genre asc, name asc",
		Filters: squirrel.Eq{"genre": genre},
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

func SongsByGenre(genre string) Options {
	return Options{
		Sort:    "genre asc, title asc",
		Filters: squirrel.Eq{"genre": genre},
	}
}

func SongsByRandom(genre string, fromYear, toYear int) Options {
	options := Options{
		Sort: "random()",
	}
	ff := squirrel.And{}
	if genre != "" {
		ff = append(ff, squirrel.Eq{"genre": genre})
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
