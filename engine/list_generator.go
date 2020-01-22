package engine

import (
	"context"
	"time"

	"github.com/cloudsonic/sonic-server/model"
)

type ListGenerator interface {
	GetNewest(ctx context.Context, offset int, size int) (Entries, error)
	GetRecent(ctx context.Context, offset int, size int) (Entries, error)
	GetFrequent(ctx context.Context, offset int, size int) (Entries, error)
	GetHighest(ctx context.Context, offset int, size int) (Entries, error)
	GetRandom(ctx context.Context, offset int, size int) (Entries, error)
	GetByName(ctx context.Context, offset int, size int) (Entries, error)
	GetByArtist(ctx context.Context, offset int, size int) (Entries, error)
	GetStarred(ctx context.Context, offset int, size int) (Entries, error)
	GetAllStarred(ctx context.Context) (artists Entries, albums Entries, mediaFiles Entries, err error)
	GetNowPlaying(ctx context.Context) (Entries, error)
	GetRandomSongs(ctx context.Context, size int, genre string) (Entries, error)
}

func NewListGenerator(ds model.DataStore, npRepo NowPlayingRepository) ListGenerator {
	return &listGenerator{ds, npRepo}
}

type listGenerator struct {
	ds     model.DataStore
	npRepo NowPlayingRepository
}

// TODO: Only return albums that have the Sort field != empty
func (g *listGenerator) query(ctx context.Context, qo model.QueryOptions, offset int, size int) (Entries, error) {
	qo.Offset = offset
	qo.Max = size
	albums, err := g.ds.Album().GetAll(qo)
	if err != nil {
		return nil, err
	}
	albumIds := make([]string, len(albums))
	for i, al := range albums {
		albumIds[i] = al.ID
	}
	annMap, err := g.ds.Annotation().GetMap(getUserID(ctx), model.AlbumItemType, albumIds)
	if err != nil {
		return nil, err
	}
	return FromAlbums(albums, annMap), err
}

func (g *listGenerator) GetNewest(ctx context.Context, offset int, size int) (Entries, error) {
	qo := model.QueryOptions{Sort: "CreatedAt", Order: "desc"}
	return g.query(ctx, qo, offset, size)
}

func (g *listGenerator) GetRecent(ctx context.Context, offset int, size int) (Entries, error) {
	qo := model.QueryOptions{Sort: "PlayDate", Order: "desc"}
	return g.query(ctx, qo, offset, size)
}

func (g *listGenerator) GetFrequent(ctx context.Context, offset int, size int) (Entries, error) {
	qo := model.QueryOptions{Sort: "PlayCount", Order: "desc"}
	return g.query(ctx, qo, offset, size)
}

func (g *listGenerator) GetHighest(ctx context.Context, offset int, size int) (Entries, error) {
	qo := model.QueryOptions{Sort: "Rating", Order: "desc"}
	return g.query(ctx, qo, offset, size)
}

func (g *listGenerator) GetByName(ctx context.Context, offset int, size int) (Entries, error) {
	qo := model.QueryOptions{Sort: "Name"}
	return g.query(ctx, qo, offset, size)
}

func (g *listGenerator) GetByArtist(ctx context.Context, offset int, size int) (Entries, error) {
	qo := model.QueryOptions{Sort: "Artist"}
	return g.query(ctx, qo, offset, size)
}

func (g *listGenerator) GetRandom(ctx context.Context, offset int, size int) (Entries, error) {
	albums, err := g.ds.Album().GetRandom(model.QueryOptions{Max: size, Offset: offset})
	if err != nil {
		return nil, err
	}

	annMap, err := g.getAnnotationsForAlbums(ctx, albums)
	if err != nil {
		return nil, err
	}
	return FromAlbums(albums, annMap), nil
}

func (g *listGenerator) getAnnotationsForAlbums(ctx context.Context, albums model.Albums) (model.AnnotationMap, error) {
	albumIds := make([]string, len(albums))
	for i, al := range albums {
		albumIds[i] = al.ID
	}
	return g.ds.Annotation().GetMap(getUserID(ctx), model.AlbumItemType, albumIds)
}

func (g *listGenerator) GetRandomSongs(ctx context.Context, size int, genre string) (Entries, error) {
	options := model.QueryOptions{Max: size}
	if genre != "" {
		options.Filters = map[string]interface{}{"genre": genre}
	}
	mediaFiles, err := g.ds.MediaFile().GetRandom(options)
	if err != nil {
		return nil, err
	}

	r := make(Entries, len(mediaFiles))
	for i, mf := range mediaFiles {
		ann, err := g.ds.Annotation().Get(getUserID(ctx), model.MediaItemType, mf.ID)
		if err != nil {
			return nil, err
		}
		r[i] = FromMediaFile(&mf, ann)
	}
	return r, nil
}

func (g *listGenerator) GetStarred(ctx context.Context, offset int, size int) (Entries, error) {
	qo := model.QueryOptions{Offset: offset, Max: size, Sort: "starred_at", Order: "desc"}
	albums, err := g.ds.Album().GetStarred(getUserID(ctx), qo)
	if err != nil {
		return nil, err
	}

	annMap, err := g.getAnnotationsForAlbums(ctx, albums)
	if err != nil {
		return nil, err
	}
	return FromAlbums(albums, annMap), nil
}

func (g *listGenerator) GetAllStarred(ctx context.Context) (artists Entries, albums Entries, mediaFiles Entries, err error) {
	options := model.QueryOptions{Sort: "starred_at", Order: "desc"}

	ars, err := g.ds.Artist().GetStarred(getUserID(ctx), options)
	if err != nil {
		return nil, nil, nil, err
	}

	als, err := g.ds.Album().GetStarred(getUserID(ctx), options)
	if err != nil {
		return nil, nil, nil, err
	}

	mfs, err := g.ds.MediaFile().GetStarred(getUserID(ctx), options)
	if err != nil {
		return nil, nil, nil, err
	}

	var mfIds []string
	for _, mf := range mfs {
		mfIds = append(mfIds, mf.ID)
	}
	trackAnnMap, err := g.ds.Annotation().GetMap(getUserID(ctx), model.MediaItemType, mfIds)
	if err != nil {
		return nil, nil, nil, err
	}

	albumAnnMap, err := g.getAnnotationsForAlbums(ctx, als)
	if err != nil {
		return nil, nil, nil, err
	}

	var artistIds []string
	for _, ar := range ars {
		artistIds = append(artistIds, ar.ID)
	}
	artistAnnMap, err := g.ds.Annotation().GetMap(getUserID(ctx), model.MediaItemType, artistIds)
	if err != nil {
		return nil, nil, nil, err
	}

	artists = FromArtists(ars, artistAnnMap)
	albums = FromAlbums(als, albumAnnMap)
	mediaFiles = FromMediaFiles(mfs, trackAnnMap)

	return
}

func (g *listGenerator) GetNowPlaying(ctx context.Context) (Entries, error) {
	npInfo, err := g.npRepo.GetAll()
	if err != nil {
		return nil, err
	}
	entries := make(Entries, len(npInfo))
	for i, np := range npInfo {
		mf, err := g.ds.MediaFile().Get(np.TrackID)
		if err != nil {
			return nil, err
		}
		ann, err := g.ds.Annotation().Get(getUserID(ctx), model.MediaItemType, mf.ID)
		entries[i] = FromMediaFile(mf, ann)
		entries[i].UserName = np.Username
		entries[i].MinutesAgo = int(time.Now().Sub(np.Start).Minutes())
		entries[i].PlayerId = np.PlayerId
		entries[i].PlayerName = np.PlayerName

	}
	return entries, nil
}
