package persistence

import (
	"context"
	"strings"
	"time"

	. "github.com/Masterminds/squirrel"
	"github.com/astaxie/beego/orm"
	"github.com/deluan/navidrome/log"
	"github.com/deluan/navidrome/model"
)

type playlist struct {
	ID        string `orm:"column(id)"`
	Name      string
	Comment   string
	Duration  float32
	Owner     string
	Public    bool
	Tracks    string
	CreatedAt time.Time
	UpdatedAt time.Time
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
		p.CreatedAt = time.Now()
	}
	p.UpdatedAt = time.Now()
	pls := r.fromModel(p)
	_, err := r.put(pls.ID, pls)
	return err
}

func (r *playlistRepository) Get(id string) (*model.Playlist, error) {
	sel := r.newSelect().Columns("*").Where(Eq{"id": id})
	var res playlist
	err := r.queryOne(sel, &res)
	pls := r.toModel(&res)
	return &pls, err
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
		ID:        p.ID,
		Name:      p.Name,
		Comment:   p.Comment,
		Duration:  p.Duration,
		Owner:     p.Owner,
		Public:    p.Public,
		CreatedAt: p.CreatedAt,
		UpdatedAt: p.UpdatedAt,
	}
	if strings.TrimSpace(p.Tracks) != "" {
		tracks := strings.Split(p.Tracks, ",")
		for _, t := range tracks {
			pls.Tracks = append(pls.Tracks, model.MediaFile{ID: t})
		}
	}
	pls.Tracks = r.loadTracks(&pls)
	return pls
}

func (r *playlistRepository) fromModel(p *model.Playlist) playlist {
	pls := playlist{
		ID:        p.ID,
		Name:      p.Name,
		Comment:   p.Comment,
		Owner:     p.Owner,
		Public:    p.Public,
		CreatedAt: p.CreatedAt,
		UpdatedAt: p.UpdatedAt,
	}
	p.Tracks = r.loadTracks(p)
	var newTracks []string
	for _, t := range p.Tracks {
		newTracks = append(newTracks, t.ID)
		pls.Duration += t.Duration
	}
	pls.Tracks = strings.Join(newTracks, ",")
	return pls
}

func (r *playlistRepository) loadTracks(p *model.Playlist) model.MediaFiles {
	mfRepo := NewMediaFileRepository(r.ctx, r.ormer)
	var ids []string
	for _, t := range p.Tracks {
		ids = append(ids, t.ID)
	}
	idsFilter := Eq{"id": ids}
	tracks, err := mfRepo.GetAll(model.QueryOptions{Filters: idsFilter})
	if err == nil {
		return tracks
	} else {
		log.Error(r.ctx, "Could not load playlist's tracks", "playlistName", p.Name, "playlistId", p.ID, err)
	}
	return nil
}

var _ model.PlaylistRepository = (*playlistRepository)(nil)
