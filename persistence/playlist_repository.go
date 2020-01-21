package persistence

import (
	"strings"

	"github.com/astaxie/beego/orm"
	"github.com/cloudsonic/sonic-server/model"
	"github.com/google/uuid"
)

type playlist struct {
	ID       string `orm:"pk;column(id)"`
	Name     string `orm:"index"`
	Comment  string
	Duration int
	Owner    string
	Public   bool
	Tracks   string `orm:"type(text)"`
}

type playlistRepository struct {
	sqlRepository
}

func NewPlaylistRepository(o orm.Ormer) model.PlaylistRepository {
	r := &playlistRepository{}
	r.ormer = o
	r.tableName = "playlist"
	return r
}

func (r *playlistRepository) Put(p *model.Playlist) error {
	if p.ID == "" {
		id, _ := uuid.NewRandom()
		p.ID = id.String()
	}
	tp := r.fromModel(p)
	err := r.put(p.ID, &tp)
	if err != nil {
		return err
	}
	return err
}

func (r *playlistRepository) Get(id string) (*model.Playlist, error) {
	tp := &playlist{ID: id}
	err := r.ormer.Read(tp)
	if err == orm.ErrNoRows {
		return nil, model.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	pls := r.toModel(tp)
	return &pls, err
}

func (r *playlistRepository) GetWithTracks(id string) (*model.Playlist, error) {
	pls, err := r.Get(id)
	if err != nil {
		return nil, err
	}
	qs := r.ormer.QueryTable(&mediaFile{})
	pls.Duration = 0
	var newTracks model.MediaFiles
	for _, t := range pls.Tracks {
		mf := &mediaFile{}
		if err := qs.Filter("id", t.ID).One(mf); err == nil {
			pls.Duration += mf.Duration
			newTracks = append(newTracks, model.MediaFile(*mf))
		}
	}
	pls.Tracks = newTracks
	return pls, err
}

func (r *playlistRepository) GetAll(options ...model.QueryOptions) (model.Playlists, error) {
	var all []playlist
	_, err := r.newQuery(options...).All(&all)
	if err != nil {
		return nil, err
	}
	return r.toModels(all)
}

func (r *playlistRepository) toModels(all []playlist) (model.Playlists, error) {
	result := make(model.Playlists, len(all))
	for i, p := range all {
		result[i] = r.toModel(&p)
	}
	return result, nil
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
