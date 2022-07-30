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
	"github.com/beego/beego/v2/client/orm"
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

func NewAlbumRepository(ctx context.Context, o orm.QueryExecutor) model.AlbumRepository {
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
		"id":              idFilter(r.tableName),
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
	sql := r.newSelectWithAnnotation("album.id")
	sql = r.withGenres(sql)
	return r.count(sql, options...)
}

func (r *albumRepository) Exists(id string) (bool, error) {
	return r.exists(Select().Where(Eq{"album.id": id}))
}

func (r *albumRepository) selectAlbum(options ...model.QueryOptions) SelectBuilder {
	sql := r.newSelectWithAnnotation("album.id", options...).Columns("album.*")
	if len(options) > 0 && options[0].Filters != nil {
		s, _, _ := options[0].Filters.ToSql()
		// If there's any reference of genre in the filter, joins with genre
		if strings.Contains(s, "genre") {
			sql = r.withGenres(sql)
			// If there's no filter on genre_id, group the results by media_file.id
			if !strings.Contains(s, "genre_id") {
				sql = sql.GroupBy("album.id")
			}
		}
	}
	return sql
}

func (r *albumRepository) Get(id string) (*model.Album, error) {
	sq := r.selectAlbum().Where(Eq{"album.id": id})
	var res model.Albums
	if err := r.queryAll(sq, &res); err != nil {
		return nil, err
	}
	if len(res) == 0 {
		return nil, model.ErrNotFound
	}
	err := r.loadAlbumGenres(&res)
	return &res[0], err
}

func (r *albumRepository) Put(m *model.Album) error {
	_, err := r.put(m.ID, m)
	if err != nil {
		return err
	}
	return r.updateGenres(m.ID, r.tableName, m.Genres)
}

func (r *albumRepository) GetAll(options ...model.QueryOptions) (model.Albums, error) {
	res, err := r.GetAllWithoutGenres(options...)
	if err != nil {
		return nil, err
	}
	err = r.loadAlbumGenres(&res)
	return res, err
}

func (r *albumRepository) GetAllWithoutGenres(options ...model.QueryOptions) (model.Albums, error) {
	sq := r.selectAlbum(options...)
	res := model.Albums{}
	err := r.queryAll(sq, &res)
	if err != nil {
		return nil, err
	}
	return res, err
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

const zwsp = string('\u200b')

type refreshAlbum struct {
	model.Album
	CurrentId      string
	SongArtists    string
	SongArtistIds  string
	AlbumArtistIds string
	GenreIds       string
	Years          string
	DiscSubtitles  string
	Comments       string
	Path           string
	MaxUpdatedAt   string
	MaxCreatedAt   string
}

func (r *albumRepository) refresh(ids ...string) error {
	stringListIds := "('" + strings.Join(ids, "','") + "')"
	var albums []refreshAlbum
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
		group_concat(f.comment, "`+zwsp+`") as comments,
		group_concat(f.mbz_album_id, ' ') as mbz_album_id, 
		group_concat(f.disc_subtitle, ' ') as disc_subtitles,
		group_concat(f.artist, ' ') as song_artists, 
		group_concat(f.artist_id, ' ') as song_artist_ids, 
		group_concat(f.album_artist_id, ' ') as album_artist_ids, 
		group_concat(f.year, ' ') as years`,
		"cf.cover_art_id", "cf.cover_art_path",
		"mfg.genre_ids").
		From("media_file f").
		LeftJoin("album a on f.album_id = a.id").
		LeftJoin(`(select album_id, id as cover_art_id, path as cover_art_path from media_file 
			where has_cover_art = true and album_id in ` + stringListIds + ` group by album_id) cf 
			on cf.album_id = f.album_id`).
		LeftJoin(`(select mf.album_id, group_concat(genre_id, ' ') as genre_ids from media_file_genres
			left join media_file mf on mf.id = media_file_id where mf.album_id in ` +
			stringListIds + ` group by mf.album_id) mfg on mfg.album_id = f.album_id`).
		Where(Eq{"f.album_id": ids}).GroupBy("f.album_id")
	err := r.queryAll(sel, &albums)
	if err != nil {
		return err
	}
	toInsert := 0
	toUpdate := 0
	for _, al := range albums {
		if al.CoverArtPath == "" || !strings.HasPrefix(conf.Server.CoverArtPriority, "embedded") {
			if path := getCoverFromPath(al.Path, al.CoverArtPath); path != "" {
				al.CoverArtId = "al-" + al.ID
				al.CoverArtPath = path
			}
		}

		if al.CoverArtId != "" {
			log.Trace(r.ctx, "Found album art", "id", al.ID, "name", al.Name, "coverArtPath", al.CoverArtPath, "coverArtId", al.CoverArtId)
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

		al.AlbumArtistID, al.AlbumArtist = getAlbumArtist(al)
		al.MinYear = getMinYear(al.Years)
		al.MbzAlbumID = getMostFrequentMbzID(r.ctx, al.MbzAlbumID, r.tableName, al.Name)
		al.Comment = getComment(al.Comments, zwsp)
		if al.CurrentId != "" {
			toUpdate++
		} else {
			toInsert++
		}
		al.AllArtistIDs = utils.SanitizeStrings(al.SongArtistIds, al.AlbumArtistID, al.ArtistID)
		al.FullText = getFullText(al.Name, al.Artist, al.AlbumArtist, al.SongArtists,
			al.SortAlbumName, al.SortArtistName, al.SortAlbumArtistName, al.DiscSubtitles)
		al.Genres = getGenres(al.GenreIds)
		err := r.Put(&al.Album)
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

func getAlbumArtist(al refreshAlbum) (id, name string) {
	if !al.Compilation {
		if al.AlbumArtist != "" {
			return al.AlbumArtistID, al.AlbumArtist
		}
		return al.ArtistID, al.Artist
	}

	ids := strings.Split(al.AlbumArtistIds, " ")
	allSame := true
	previous := al.AlbumArtistID
	for _, id := range ids {
		if id == previous {
			continue
		}
		allSame = false
		break
	}
	if allSame {
		return al.AlbumArtistID, al.AlbumArtist
	}
	return consts.VariousArtistsID, consts.VariousArtists
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
