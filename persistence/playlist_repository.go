package persistence

import (
	"context"
	"strings"

	. "github.com/Masterminds/squirrel"
	"github.com/astaxie/beego/orm"
	"github.com/deluan/navidrome/model"
	"github.com/google/uuid"
)

type playlist struct {
	ID       string `orm:"column(id)"`
	Name     string
	Comment  string
	Duration int
	Owner    string
	Public   bool
	Tracks   string
}

type playlistRepository struct {
	sqlRepository
}

func NewPlaylistRepository(ctx context.Context, o orm.Ormer) model.PlaylistRepository {
	r := &playlistRepository{}
	r.ctx = ctx
	r.ormer = o
	r.tableName = "playlist"
	return r
}

func (r *playlistRepository) CountAll() (int64, error) {
	return r.count(Select())
}

func (r *playlistRepository) Exists(id string) (bool, error) {
	return r.exists(Select().Where(Eq{"id": id}))
}

func (r *playlistRepository) Delete(id string) error {
	return r.delete(Eq{"id": id})
}

func (r *playlistRepository) Put(p *model.Playlist) error {
	if p.ID == "" {
		id, _ := uuid.NewRandom()
		p.ID = id.String()
	}
	values, _ := toSqlArgs(r.fromModel(p))
	update := Update(r.tableName).Where(Eq{"id": p.ID}).SetMap(values)
	count, err := r.executeSQL(update)
	if err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	insert := Insert(r.tableName).SetMap(values)
	_, err = r.executeSQL(insert)
	return err
}

func (r *playlistRepository) Get(id string) (*model.Playlist, error) {
	sel := r.newSelect().Columns("*").Where(Eq{"id": id})
	var res playlist
	err := r.queryOne(sel, &res)
	pls := r.toModel(&res)
	return &pls, err
}

func (r *playlistRepository) GetWithTracks(id string) (*model.Playlist, error) {
	pls, err := r.Get(id)
	if err != nil {
		return nil, err
	}
	mfRepo := NewMediaFileRepository(r.ctx, r.ormer)
	pls.Duration = 0
	var newTracks model.MediaFiles
	for _, t := range pls.Tracks {
		mf, err := mfRepo.Get(t.ID)
		if err != nil {
			continue
		}
		pls.Duration += mf.Duration
		newTracks = append(newTracks, model.MediaFile(*mf))
	}
	pls.Tracks = newTracks
	return pls, err
}

func (r *playlistRepository) GetAll(options ...model.QueryOptions) (model.Playlists, error) {
	sel := r.newSelect(options...).Columns("*")
	var res []playlist
	err := r.queryAll(sel, &res)
	return r.toModels(res), err
}

func (r *playlistRepository) toModels(all []playlist) model.Playlists {
	result := make(model.Playlists, len(all))
	for i, p := range all {
		result[i] = r.toModel(&p)
	}
	return result
}

func (r *playlistRepository) toModel(p *playlist) model.Playlist {
	pls := model.Playlist{
		ID:       p.ID,
		Name:     p.Name,
		Comment:  p.Comment,
		Duration: p.Duration,
		Owner:    p.Owner,
		Public:   p.Public,
	}
	if strings.TrimSpace(p.Tracks) != "" {
		tracks := strings.Split(p.Tracks, ",")
		for _, t := range tracks {
			pls.Tracks = append(pls.Tracks, model.MediaFile{ID: t})
		}
	}
	return pls
}

func (r *playlistRepository) fromModel(p *model.Playlist) playlist {
	pls := playlist{
		ID:       p.ID,
		Name:     p.Name,
		Comment:  p.Comment,
		Duration: p.Duration,
		Owner:    p.Owner,
		Public:   p.Public,
	}
	var newTracks []string
	for _, t := range p.Tracks {
		newTracks = append(newTracks, t.ID)
	}
	pls.Tracks = strings.Join(newTracks, ",")
	return pls
}

var _ model.PlaylistRepository = (*playlistRepository)(nil)
