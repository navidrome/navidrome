package persistence

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	. "github.com/Masterminds/squirrel"
	"github.com/astaxie/beego/orm"
	"github.com/deluan/rest"
	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/utils"
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
		"name":           "order_album_name asc, order_album_artist_name asc",
		"artist":         "compilation asc, order_album_artist_name asc, order_album_name asc",
		"random":         "RANDOM()",
		"max_year":       "max_year asc, name, order_album_name asc",
		"recently_added": recentlyAddedSort(),
	}
	r.filterMappings = map[string]filterFunc{
		"name":            fullTextFilter,
		"compilation":     booleanFilter,
		"artist_id":       artistFilter,
		"year":            yearFilter,
		"recently_played": recentlyPlayedFilter,
		"starred":         booleanFilter,
		"has_rating":      hasRatingFilter,
	}

	return r
}

func recentlyAddedSort() string {
	if conf.Server.RecentlyAddedByModTime {
		return "updated_at"
	}
	return "created_at"
}

func recentlyPlayedFilter(field string, value interface{}) Sqlizer {
	return Gt{"play_count": 0}
}

func hasRatingFilter(field string, value interface{}) Sqlizer {
	return Gt{"rating": 0}
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
	return Like{"all_artist_ids": fmt.Sprintf("%%%s%%", value)}
}

func (r *albumRepository) CountAll(options ...model.QueryOptions) (int64, error) {
	return r.count(r.selectAlbum(), options...)
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

// Return a map of mediafiles that have embedded covers for the given album ids
func (r *albumRepository) getEmbeddedCovers(ids []string) (map[string]model.MediaFile, error) {
	var mfs model.MediaFiles
	coverSql := Select("album_id", "id", "path").Distinct().From("media_file").
		Where(And{Eq{"has_cover_art": true}, Eq{"album_id": ids}}).
		GroupBy("album_id")
	err := r.queryAll(coverSql, &mfs)
	if err != nil {
		return nil, err
	}

	result := map[string]model.MediaFile{}
	for _, mf := range mfs {
		result[mf.AlbumID] = mf
	}
	return result, nil
}

func (r *albumRepository) Refresh(ids ...string) error {
	chunks := utils.BreakUpStringSlice(ids, 100)
	for _, chunk := range chunks {
		err := r.refresh(chunk...)
		if err != nil {
			return err
		}
	}
	return nil
}

func (r *albumRepository) refresh(ids ...string) error {
	type refreshAlbum struct {
		model.Album
		CurrentId     string
		SongArtists   string
		SongArtistIds string
		Years         string
		DiscSubtitles string
		Comments      string
		Path          string
		MaxUpdatedAt  string
		MaxCreatedAt  string
	}
	var albums []refreshAlbum
	const zwsp = string('\u200b')
	sel := Select(`f.album_id as id, f.album as name, f.artist, f.album_artist, f.artist_id, f.album_artist_id, 
		f.sort_album_name, f.sort_artist_name, f.sort_album_artist_name, f.order_album_name, f.order_album_artist_name, 
		f.path, f.mbz_album_artist_id, f.mbz_album_type, f.mbz_album_comment, f.catalog_num, f.compilation, f.genre, 
		count(f.id) as song_count,  
		sum(f.duration) as duration,
		sum(f.size) as size,
		max(f.year) as max_year, 
		max(f.updated_at) as max_updated_at,
		max(f.created_at) as max_created_at,
		a.id as current_id,  
		group_concat(f.comment, "` + zwsp + `") as comments,
		group_concat(f.mbz_album_id, ' ') as mbz_album_id, 
		group_concat(f.disc_subtitle, ' ') as disc_subtitles,
		group_concat(f.artist, ' ') as song_artists, 
		group_concat(f.artist_id, ' ') as song_artist_ids, 
		group_concat(f.year, ' ') as years`).
		From("media_file f").
		LeftJoin("album a on f.album_id = a.id").
		Where(Eq{"f.album_id": ids}).GroupBy("f.album_id")
	err := r.queryAll(sel, &albums)
	if err != nil {
		return err
	}

	covers, err := r.getEmbeddedCovers(ids)
	if err != nil {
		return nil
	}

	toInsert := 0
	toUpdate := 0
	for _, al := range albums {
		embedded, hasCoverArt := covers[al.ID]
		if hasCoverArt {
			al.CoverArtId = embedded.ID
			al.CoverArtPath = embedded.Path
		}

		if !hasCoverArt || !strings.HasPrefix(conf.Server.CoverArtPriority, "embedded") {
			if path := getCoverFromPath(al.Path, al.CoverArtPath); path != "" {
				al.CoverArtId = "al-" + al.ID
				al.CoverArtPath = path
			}
		}

		if al.CoverArtId != "" {
			log.Trace(r.ctx, "Found album art", "id", al.ID, "name", al.Name, "coverArtPath", al.CoverArtPath, "coverArtId", al.CoverArtId, "hasCoverArt", hasCoverArt)
		} else {
			log.Trace(r.ctx, "Could not find album art", "id", al.ID, "name", al.Name)
		}

		// Somehow, beego cannot parse the datetimes for the query above
		if al.UpdatedAt, err = time.Parse(time.RFC3339Nano, al.MaxUpdatedAt); err != nil {
			al.UpdatedAt = time.Now()
		}
		if al.CreatedAt, err = time.Parse(time.RFC3339Nano, al.MaxCreatedAt); err != nil {
			al.CreatedAt = al.UpdatedAt
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
		al.MbzAlbumID = getMbzId(r.ctx, al.MbzAlbumID, r.tableName, al.Name)
		al.Comment = getComment(al.Comments, zwsp)
		if al.CurrentId != "" {
			toUpdate++
		} else {
			toInsert++
		}
		al.AllArtistIDs = utils.SanitizeStrings(al.SongArtistIds, al.AlbumArtistID, al.ArtistID)
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

func getComment(comments string, separator string) string {
	cs := strings.Split(comments, separator)
	if len(cs) == 0 {
		return ""
	}
	first := cs[0]
	for _, c := range cs[1:] {
		if first != c {
			return ""
		}
	}
	return first
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
func getCoverFromPath(mediaPath string, embeddedPath string) string {
	n, err := os.Open(filepath.Dir(mediaPath))
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
			if embeddedPath != "" {
				return ""
			}
			continue
		}

		for _, name := range names {
			match, _ := filepath.Match(pat, strings.ToLower(name))
			if match && utils.IsImageFile(name) {
				return filepath.Join(filepath.Dir(mediaPath), name)
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

func (r albumRepository) Delete(id string) error {
	return r.delete(Eq{"id": id})
}

func (r albumRepository) Save(entity interface{}) (string, error) {
	album := entity.(*model.Album)
	id, err := r.put(album.ID, album)
	return id, err
}

func (r albumRepository) Update(entity interface{}, cols ...string) error {
	album := entity.(*model.Album)
	_, err := r.put(album.ID, album)
	return err
}

var _ model.AlbumRepository = (*albumRepository)(nil)
var _ model.ResourceRepository = (*albumRepository)(nil)
var _ rest.Persistable = (*albumRepository)(nil)
