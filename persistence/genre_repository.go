package persistence

import (
	"strconv"

	"github.com/astaxie/beego/orm"
	"github.com/cloudsonic/sonic-server/model"
)

type genreRepository struct{}

func NewGenreRepository() model.GenreRepository {
	return &genreRepository{}
}

func (r genreRepository) GetAll() (model.Genres, error) {
	o := Db()
	genres := make(map[string]model.Genre)

	// Collect SongCount
	var res []orm.Params
	_, err := o.Raw("select genre, count(*) as c from media_file group by genre").Values(&res)
	if err != nil {
		return nil, err
	}
	for _, r := range res {
		name := r["genre"].(string)
		count := r["c"].(string)
		g, ok := genres[name]
		if !ok {
			g = model.Genre{Name: name}
		}
		g.SongCount, _ = strconv.Atoi(count)
		genres[name] = g
	}

	// Collect AlbumCount
	_, err = o.Raw("select genre, count(*) as c from album group by genre").Values(&res)
	if err != nil {
		return nil, err
	}
	for _, r := range res {
		name := r["genre"].(string)
		count := r["c"].(string)
		g, ok := genres[name]
		if !ok {
			g = model.Genre{Name: name}
		}
		g.AlbumCount, _ = strconv.Atoi(count)
		genres[name] = g
	}

	// Build response
	result := model.Genres{}
	for _, g := range genres {
		result = append(result, g)
	}
	return result, err
}
