package engine

import (
	"context"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/deluan/navidrome/model"
)

type ListGenerator interface {
	GetAllStarred(ctx context.Context) (artists Entries, albums Entries, mediaFiles Entries, err error)
	GetNowPlaying(ctx context.Context) (Entries, error)
	GetSongs(ctx context.Context, offset, size int, filter ListFilter) (Entries, error)
	GetAlbums(ctx context.Context, offset, size int, filter ListFilter) (Entries, error)
}

func NewListGenerator(ds model.DataStore, npRepo NowPlayingRepository) ListGenerator {
	return &listGenerator{ds, npRepo}
}

type ListFilter model.QueryOptions

func ByNewest() ListFilter {
	return ListFilter{Sort: "createdAt", Order: "desc"}
}

func ByRecent() ListFilter {
	return ListFilter{Sort: "playDate", Order: "desc", Filters: squirrel.Gt{"play_date": time.Time{}}}
}

func ByFrequent() ListFilter {
	return ListFilter{Sort: "playCount", Order: "desc", Filters: squirrel.Gt{"play_count": 0}}
}

func ByRandom() ListFilter {
	return ListFilter{Sort: "random()"}
}

func ByName() ListFilter {
	return ListFilter{Sort: "name"}
}

func ByArtist() ListFilter {
	return ListFilter{Sort: "artist"}
}

func ByStarred() ListFilter {
	return ListFilter{Sort: "starred_at", Order: "desc", Filters: squirrel.Eq{"starred": true}}
}

func ByRating() ListFilter {
	return ListFilter{Sort: "Rating", Order: "desc", Filters: squirrel.Gt{"rating": 0}}
}

func ByGenre(genre string) ListFilter {
	return ListFilter{
		Sort:    "genre asc, name asc",
		Filters: squirrel.Eq{"genre": genre},
	}
}

func ByYear(fromYear, toYear int) ListFilter {
	sortOption := "max_year, name"
	if fromYear > toYear {
		fromYear, toYear = toYear, fromYear
		sortOption = "max_year desc, name"
	}
	return ListFilter{
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

func SongsByGenre(genre string) ListFilter {
	return ListFilter{
		Sort:    "genre asc, title asc",
		Filters: squirrel.Eq{"genre": genre},
	}
}

func SongsByRandom(genre string, fromYear, toYear int) ListFilter {
	options := ListFilter{
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

type listGenerator struct {
	ds     model.DataStore
	npRepo NowPlayingRepository
}

func (g *listGenerator) GetSongs(ctx context.Context, offset, size int, filter ListFilter) (Entries, error) {
	qo := model.QueryOptions(filter)
	qo.Offset = offset
	qo.Max = size
	mediaFiles, err := g.ds.MediaFile(ctx).GetAll(qo)
	if err != nil {
		return nil, err
	}

	return FromMediaFiles(mediaFiles), nil
}

func (g *listGenerator) GetAlbums(ctx context.Context, offset, size int, filter ListFilter) (Entries, error) {
	qo := model.QueryOptions(filter)
	qo.Offset = offset
	qo.Max = size
	albums, err := g.ds.Album(ctx).GetAll(qo)
	if err != nil {
		return nil, err
	}

	return FromAlbums(albums), nil
}

func (g *listGenerator) GetStarred(ctx context.Context, offset int, size int) (Entries, error) {
	qo := model.QueryOptions{Offset: offset, Max: size, Sort: "starred_at", Order: "desc"}
	albums, err := g.ds.Album(ctx).GetStarred(qo)
	if err != nil {
		return nil, err
	}

	return FromAlbums(albums), nil
}

func (g *listGenerator) GetAllStarred(ctx context.Context) (artists Entries, albums Entries, mediaFiles Entries, err error) {
	options := model.QueryOptions{Sort: "starred_at", Order: "desc"}

	ars, err := g.ds.Artist(ctx).GetStarred(options)
	if err != nil {
		return nil, nil, nil, err
	}

	als, err := g.ds.Album(ctx).GetStarred(options)
	if err != nil {
		return nil, nil, nil, err
	}

	mfs, err := g.ds.MediaFile(ctx).GetStarred(options)
	if err != nil {
		return nil, nil, nil, err
	}

	artists = FromArtists(ars)
	albums = FromAlbums(als)
	mediaFiles = FromMediaFiles(mfs)

	return
}

func (g *listGenerator) GetNowPlaying(ctx context.Context) (Entries, error) {
	npInfo, err := g.npRepo.GetAll()
	if err != nil {
		return nil, err
	}
	entries := make(Entries, len(npInfo))
	for i, np := range npInfo {
		mf, err := g.ds.MediaFile(ctx).Get(np.TrackID)
		if err != nil {
			return nil, err
		}
		entries[i] = FromMediaFile(mf)
		entries[i].UserName = np.Username
		entries[i].MinutesAgo = int(time.Since(np.Start).Minutes())
		entries[i].PlayerId = np.PlayerId
		entries[i].PlayerName = np.PlayerName
	}
	return entries, nil
}
