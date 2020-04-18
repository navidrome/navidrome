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
	GetRandomSongs(ctx context.Context, size int, genre string) (Entries, error)
	GetAlbums(ctx context.Context, offset, size int, filter AlbumFilter) (Entries, error)
}

func NewListGenerator(ds model.DataStore, npRepo NowPlayingRepository) ListGenerator {
	return &listGenerator{ds, npRepo}
}

type AlbumFilter model.QueryOptions

func ByNewest() AlbumFilter {
	return AlbumFilter{Sort: "createdAt", Order: "desc"}
}

func ByRecent() AlbumFilter {
	return AlbumFilter{Sort: "playDate", Order: "desc", Filters: squirrel.Gt{"play_date": time.Time{}}}
}

func ByFrequent() AlbumFilter {
	return AlbumFilter{Sort: "playCount", Order: "desc", Filters: squirrel.Gt{"play_count": 0}}
}

func ByRandom() AlbumFilter {
	return AlbumFilter{Sort: "random()"}
}

func ByName() AlbumFilter {
	return AlbumFilter{Sort: "name"}
}

func ByArtist() AlbumFilter {
	return AlbumFilter{Sort: "artist"}
}

func ByStarred() AlbumFilter {
	return AlbumFilter{Sort: "starred_at", Order: "desc", Filters: squirrel.Eq{"starred": true}}
}

func ByRating() AlbumFilter {
	return AlbumFilter{Sort: "Rating", Order: "desc", Filters: squirrel.Gt{"rating": 0}}
}

func ByGenre(genre string) AlbumFilter {
	return AlbumFilter{
		Sort:    "genre asc, name asc",
		Filters: squirrel.Eq{"genre": genre},
	}
}

func ByYear(fromYear, toYear int) AlbumFilter {
	return AlbumFilter{
		Sort: "max_year, name",
		Filters: squirrel.Or{
			squirrel.And{
				squirrel.LtOrEq{"min_year": toYear},
				squirrel.GtOrEq{"max_year": toYear},
			},
			squirrel.And{
				squirrel.LtOrEq{"min_year": fromYear},
				squirrel.GtOrEq{"max_year": fromYear},
			},
		},
	}
}

type listGenerator struct {
	ds     model.DataStore
	npRepo NowPlayingRepository
}

func (g *listGenerator) query(ctx context.Context, qo model.QueryOptions) (Entries, error) {
	albums, err := g.ds.Album(ctx).GetAll(qo)
	if err != nil {
		return nil, err
	}
	albumIds := make([]string, len(albums))
	for i, al := range albums {
		albumIds[i] = al.ID
	}
	return FromAlbums(albums), err
}

func (g *listGenerator) GetRandomSongs(ctx context.Context, size int, genre string) (Entries, error) {
	options := model.QueryOptions{Max: size}
	if genre != "" {
		options.Filters = squirrel.Eq{"genre": genre}
	}
	mediaFiles, err := g.ds.MediaFile(ctx).GetRandom(options)
	if err != nil {
		return nil, err
	}

	return FromMediaFiles(mediaFiles), nil
}

func (g *listGenerator) GetAlbums(ctx context.Context, offset, size int, filter AlbumFilter) (Entries, error) {
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

	var mfIds []string
	for _, mf := range mfs {
		mfIds = append(mfIds, mf.ID)
	}

	var artistIds []string
	for _, ar := range ars {
		artistIds = append(artistIds, ar.ID)
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
		entries[i].MinutesAgo = int(time.Now().Sub(np.Start).Minutes())
		entries[i].PlayerId = np.PlayerId
		entries[i].PlayerName = np.PlayerName

	}
	return entries, nil
}
