package persistence

import (
	"context"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	. "github.com/Masterminds/squirrel"
	"github.com/astaxie/beego/orm"
	"github.com/deluan/navidrome/conf"
	"github.com/deluan/navidrome/consts"
	"github.com/deluan/navidrome/log"
	"github.com/deluan/navidrome/model"
	"github.com/deluan/navidrome/utils"
	"github.com/deluan/rest"
)

type albumRepository struct {
	sqlRepository
	sqlRestful
}

func NewAlbumRepository(ctx context.Context, o orm.Ormer) model.AlbumRepository {
	r := &albumRepository{}
	r.ctx = ctx
	r.ormer = o
	r.tableName = "album"
	r.sortMappings = map[string]string{
		"name":     "order_album_name",
		"artist":   "compilation asc, order_album_artist_name asc, order_album_name asc",
		"random":   "RANDOM()",
		"max_year": "max_year asc, name, order_album_name asc",
	}
	r.filterMappings = map[string]filterFunc{
		"name":        fullTextFilter,
		"compilation": booleanFilter,
		"artist_id":   artistFilter,
		"year":        yearFilter,
	}

	return r
}

func yearFilter(field string, value interface{}) Sqlizer {
	return Or{
		And{
			Gt{"min_year": 0},
			LtOrEq{"min_year": value},
			GtOrEq{"max_year": value},
		},
		Eq{"max_year": value},
	}
}

func artistFilter(field string, value interface{}) Sqlizer {
	return exists("media_file", And{
		ConcatExpr("album_id=album.id"),
		Or{
			Eq{"artist_id": value},
			Eq{"album_artist_id": value},
		},
	})
}

func (r *albumRepository) CountAll(options ...model.QueryOptions) (int64, error) {
	return r.count(Select(), options...)
}

func (r *albumRepository) Exists(id string) (bool, error) {
	return r.exists(Select().Where(Eq{"id": id}))
}

func (r *albumRepository) selectAlbum(options ...model.QueryOptions) SelectBuilder {
	return r.newSelectWithAnnotation("album.id", options...).Columns("*")
}

func (r *albumRepository) Get(id string) (*model.Album, error) {
	sq := r.selectAlbum().Where(Eq{"id": id})
	var res model.Albums
	if err := r.queryAll(sq, &res); err != nil {
		return nil, err
	}
	if len(res) == 0 {
		return nil, model.ErrNotFound
	}
	return &res[0], nil
}

func (r *albumRepository) FindByArtist(artistId string) (model.Albums, error) {
	sq := r.selectAlbum().Where(Eq{"album_artist_id": artistId}).OrderBy("max_year")
	res := model.Albums{}
	err := r.queryAll(sq, &res)
	return res, err
}

func (r *albumRepository) GetAll(options ...model.QueryOptions) (model.Albums, error) {
	sq := r.selectAlbum(options...)
	res := model.Albums{}
	err := r.queryAll(sq, &res)
	return res, err
}

// TODO Keep order when paginating
func (r *albumRepository) GetRandom(options ...model.QueryOptions) (model.Albums, error) {
	sq := r.selectAlbum(options...)
	sq = sq.OrderBy("RANDOM()")
	results := model.Albums{}
	err := r.queryAll(sq, &results)
	return results, err
}

func (r *albumRepository) Refresh(ids ...string) error {
	type refreshAlbum struct {
		model.Album
		CurrentId     string
		HasCoverArt   bool
		SongArtists   string
		Years         string
		DiscSubtitles string
	}
	var albums []refreshAlbum
	sel := Select(`f.album_id as id, f.album as name, f.artist, f.album_artist, f.artist_id, f.album_artist_id, 
		f.sort_album_name, f.sort_artist_name, f.sort_album_artist_name,
		f.order_album_name, f.order_album_artist_name,
		f.compilation, f.genre, max(f.year) as max_year, sum(f.duration) as duration, 
		count(*) as song_count, a.id as current_id, 
		f2.id as cover_art_id, f2.path as cover_art_path, f2.has_cover_art, 
		group_concat(f.disc_subtitle, ' ') as disc_subtitles,
		group_concat(f.artist, ' ') as song_artists, group_concat(f.year, ' ') as years`).
		From("media_file f").
		LeftJoin("album a on f.album_id = a.id").
		LeftJoin("(select * from media_file where has_cover_art) f2 on (f.album_id = f2.album_id)").
		Where(Eq{"f.album_id": ids}).GroupBy("f.album_id")
	err := r.queryAll(sel, &albums)
	if err != nil {
		return err
	}

	toInsert := 0
	toUpdate := 0
	for _, al := range albums {
		if !al.HasCoverArt || !strings.HasPrefix(conf.Server.CoverArtPriority, "embedded") {
			if path := getCoverFromPath(al.CoverArtPath, al.HasCoverArt); path != "" {
				al.CoverArtId = "al-" + al.ID
				al.CoverArtPath = path
			} else if !al.HasCoverArt {
				al.CoverArtId = ""
			}
		}

		if al.Compilation {
			al.AlbumArtist = consts.VariousArtists
			al.AlbumArtistID = consts.VariousArtistsID
		}
		if al.AlbumArtist == "" {
			al.AlbumArtist = al.Artist
			al.AlbumArtistID = al.ArtistID
		}
		al.MinYear = getMinYear(al.Years)
		al.UpdatedAt = time.Now()
		if al.CurrentId != "" {
			toUpdate++
		} else {
			toInsert++
			al.CreatedAt = time.Now()
		}
		al.FullText = getFullText(al.Name, al.Artist, al.AlbumArtist, al.SongArtists,
			al.SortAlbumName, al.SortArtistName, al.SortAlbumArtistName, al.DiscSubtitles)
		_, err := r.put(al.ID, al.Album)
		if err != nil {
			return err
		}
	}
	if toInsert > 0 {
		log.Debug(r.ctx, "Inserted new albums", "totalInserted", toInsert)
	}
	if toUpdate > 0 {
		log.Debug(r.ctx, "Updated albums", "totalUpdated", toUpdate)
	}
	return err
}

func getMinYear(years string) int {
	ys := strings.Fields(years)
	sort.Strings(ys)
	for _, y := range ys {
		if y != "0" {
			r, _ := strconv.Atoi(y)
			return r
		}
	}
	return 0
}

// GetCoverFromPath accepts a path to a file, and returns a path to an eligible cover image from the
// file's directory (as configured with CoverArtPriority). If no cover file is found, among
// available choices, or an error occurs, an empty string is returned. If HasEmbeddedCover is true,
// and 'embedded' is matched among eligible choices, GetCoverFromPath will return early with an
// empty path.
func getCoverFromPath(path string, hasEmbeddedCover bool) string {
	n, err := os.Open(filepath.Dir(path))
	if err != nil {
		return ""
	}

	defer n.Close()
	names, err := n.Readdirnames(-1)
	if err != nil {
		return ""
	}

	for _, p := range strings.Split(conf.Server.CoverArtPriority, ",") {
		pat := strings.ToLower(strings.TrimSpace(p))
		if pat == "embedded" {
			if hasEmbeddedCover {
				return ""
			}
			continue
		}

		for _, name := range names {
			match, _ := filepath.Match(pat, strings.ToLower(name))
			if match && utils.IsImageFile(name) {
				return filepath.Join(filepath.Dir(path), name)
			}
		}
	}

	return ""
}

func (r *albumRepository) purgeEmpty() error {
	del := Delete(r.tableName).Where("id not in (select distinct(album_id) from media_file)")
	c, err := r.executeSQL(del)
	if err == nil {
		if c > 0 {
			log.Debug(r.ctx, "Purged empty albums", "totalDeleted", c)
		}
	}
	return err
}

func (r *albumRepository) GetStarred(options ...model.QueryOptions) (model.Albums, error) {
	sq := r.selectAlbum(options...).Where("starred = true")
	starred := model.Albums{}
	err := r.queryAll(sq, &starred)
	return starred, err
}

func (r *albumRepository) Search(q string, offset int, size int) (model.Albums, error) {
	results := model.Albums{}
	err := r.doSearch(q, offset, size, &results, "name")
	return results, err
}

func (r *albumRepository) Count(options ...rest.QueryOptions) (int64, error) {
	return r.CountAll(r.parseRestOptions(options...))
}

func (r *albumRepository) Read(id string) (interface{}, error) {
	return r.Get(id)
}

func (r *albumRepository) ReadAll(options ...rest.QueryOptions) (interface{}, error) {
	return r.GetAll(r.parseRestOptions(options...))
}

func (r *albumRepository) EntityName() string {
	return "album"
}

func (r *albumRepository) NewInstance() interface{} {
	return &model.Album{}
}

var _ model.AlbumRepository = (*albumRepository)(nil)
var _ model.ResourceRepository = (*albumRepository)(nil)
