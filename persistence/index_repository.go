package persistence

import (
	"sort"
	"strings"

	"github.com/astaxie/beego/orm"
	"github.com/cloudsonic/sonic-server/conf"
	"github.com/cloudsonic/sonic-server/model"
	"github.com/cloudsonic/sonic-server/utils"
)

type ArtistInfo struct {
	ID         int `orm:"pk;auto;column(id)"`
	Idx        string
	ArtistID   string `orm:"column(artist_id)"`
	Artist     string
	AlbumCount int
}

type tempIndex map[string]model.ArtistInfo

type artistIndexRepository struct {
	sqlRepository
}

func NewArtistIndexRepository() model.ArtistIndexRepository {
	r := &artistIndexRepository{}
	r.tableName = "artist_info"
	return r
}

func (r *artistIndexRepository) Put(idx *model.ArtistIndex) error {
	return withTx(func(o orm.Ormer) error {
		_, err := r.newQuery(o).Filter("idx", idx.ID).Delete()
		if err != nil {
			return err
		}
		for _, artist := range idx.Artists {
			a := ArtistInfo{
				Idx:        idx.ID,
				ArtistID:   artist.ArtistID,
				Artist:     artist.Artist,
				AlbumCount: artist.AlbumCount,
			}
			err := r.insert(o, &a)
			if err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *artistIndexRepository) Refresh() error {
	o := Db()

	indexGroups := utils.ParseIndexGroups(conf.Sonic.IndexGroups)
	artistIndex := make(map[string]tempIndex)

	var artists []Artist
	_, err := o.QueryTable(&Artist{}).All(&artists)
	if err != nil {
		return err
	}

	for _, ar := range artists {
		r.collectIndex(indexGroups, &ar, artistIndex)
	}

	return r.saveIndex(artistIndex)
}

func (r *artistIndexRepository) collectIndex(ig utils.IndexGroups, a *Artist, artistIndex map[string]tempIndex) {
	name := a.Name
	indexName := strings.ToLower(utils.NoArticle(name))
	if indexName == "" {
		return
	}
	group := r.findGroup(ig, indexName)
	artists := artistIndex[group]
	if artists == nil {
		artists = make(tempIndex)
		artistIndex[group] = artists
	}
	artists[indexName] = model.ArtistInfo{ArtistID: a.ID, Artist: a.Name, AlbumCount: a.AlbumCount}
}

func (r *artistIndexRepository) findGroup(ig utils.IndexGroups, name string) string {
	for k, v := range ig {
		key := strings.ToLower(k)
		if strings.HasPrefix(name, key) {
			return v
		}
	}
	return "#"
}

func (r *artistIndexRepository) saveIndex(artistIndex map[string]tempIndex) error {
	r.DeleteAll()
	for k, temp := range artistIndex {
		idx := &model.ArtistIndex{ID: k}
		for _, v := range temp {
			idx.Artists = append(idx.Artists, v)
		}
		err := r.Put(idx)
		if err != nil {
			return err
		}
	}

	return nil
}

func (r *artistIndexRepository) GetAll() (model.ArtistIndexes, error) {
	var all []ArtistInfo
	_, err := r.newQuery(Db()).OrderBy("idx", "artist").All(&all)
	if err != nil {
		return nil, err
	}

	fullIdx := make(map[string]*model.ArtistIndex)
	for _, a := range all {
		idx, ok := fullIdx[a.Idx]
		if !ok {
			idx = &model.ArtistIndex{ID: a.Idx}
			fullIdx[a.Idx] = idx
		}
		idx.Artists = append(idx.Artists, model.ArtistInfo{
			ArtistID:   a.ArtistID,
			Artist:     a.Artist,
			AlbumCount: a.AlbumCount,
		})
	}
	var result model.ArtistIndexes
	for _, idx := range fullIdx {
		result = append(result, *idx)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].ID < result[j].ID
	})
	return result, nil
}

var _ model.ArtistIndexRepository = (*artistIndexRepository)(nil)
