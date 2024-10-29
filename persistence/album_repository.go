package persistence

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"slices"
	"time"

	. "github.com/Masterminds/squirrel"
	"github.com/deluan/rest"
	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/utils/slice"
	"github.com/pocketbase/dbx"
)

type albumRepository struct {
	sqlRepository
}

type dbAlbum struct {
	*model.Album   `structs:",flatten"`
	Discs          string `structs:"-" json:"discs"`
	ParticipantIDs string `structs:"-" json:"-"`
	TagIds         string `structs:"-" json:"-"`
	parsedTagIDs   []string
}

func (a *dbAlbum) PostScan() error {
	if a.Discs != "" {
		if err := json.Unmarshal([]byte(a.Discs), &a.Album.Discs); err != nil {
			return err
		}
	}
	a.Album.Participations = parseParticipations(a.ParticipantIDs)
	err := json.Unmarshal([]byte(a.TagIds), &a.parsedTagIDs)
	if err != nil {
		return fmt.Errorf("error parsing album tags: %w", err)
	}
	return nil
}

func (a *dbAlbum) PostMapArgs(args map[string]any) error {
	fullText := []string{a.Name, a.SortAlbumName, a.AlbumArtist}
	fullText = append(fullText, a.Album.Participations.AllNames()...)
	fullText = append(fullText, slices.Collect(maps.Values(a.Album.Discs))...)
	args["full_text"] = formatFullText(fullText...)

	args["tag_ids"] = buildTagIDs(a.Album.Tags)
	args["participant_ids"] = buildParticipantIDs(a.Album.Participations)
	delete(args, "tags")
	delete(args, "participations")
	if len(a.Album.Discs) == 0 {
		args["discs"] = "{}"
		return nil
	}
	b, err := json.Marshal(a.Album.Discs)
	if err != nil {
		return err
	}
	args["discs"] = string(b)
	return nil
}

func (a *dbAlbum) tagIDs() []string {
	return a.parsedTagIDs
}

type dbAlbums []dbAlbum

func (as dbAlbums) tagIDs() []string {
	var ids []string
	for _, mf := range as {
		ids = append(ids, mf.parsedTagIDs...)
	}
	return slice.Unique(ids)
}

func (as dbAlbums) setTags(tagMap map[string]model.Tag) {
	for i, mf := range as {
		tags := model.Tags{}
		for _, id := range mf.parsedTagIDs {
			if tag, ok := tagMap[id]; ok {
				tags[tag.TagName] = append(tags[tag.TagName], tag.TagValue)
			}
		}
		as[i].Album.Tags = tags
	}
}

func (as dbAlbums) toModels() model.Albums {
	return slice.Map(as, func(a dbAlbum) model.Album { return *a.Album })
}

func (as dbAlbums) getParticipantIDs() []string {
	var ids []string
	for _, a := range as {
		ids = append(ids, a.Participations.AllIDs()...)
	}
	return slice.Unique(ids)
}

func (as dbAlbums) setParticipations(participantMap map[string]string) {
	for i, a := range as {
		for role, artists := range a.Album.Participations {
			for j, artist := range artists {
				if name, ok := participantMap[artist.ID]; ok {
					as[i].Album.Participations[role][j].Name = name
				}
			}
		}
	}
}

func NewAlbumRepository(ctx context.Context, db dbx.Builder) model.AlbumRepository {
	r := &albumRepository{}
	r.ctx = ctx
	r.db = db
	r.tableName = "album"
	r.registerModel(&model.Album{}, map[string]filterFunc{
		"id":              idFilter(r.tableName),
		"name":            fullTextFilter(r.tableName),
		"compilation":     booleanFilter,
		"artist_id":       artistFilter,
		"year":            yearFilter,
		"recently_played": recentlyPlayedFilter,
		"starred":         booleanFilter,
		"has_rating":      hasRatingFilter,
		"genre_id":        tagIDFilter,
	})
	r.setSortMappings(map[string]string{
		"name":           "order_album_name, order_album_artist_name",
		"artist":         "compilation, order_album_artist_name, order_album_name",
		"album_artist":   "compilation, order_album_artist_name, order_album_name",
		"max_year":       "coalesce(nullif(original_date,''), cast(max_year as text)), release_date, name",
		"random":         "random",
		"recently_added": recentlyAddedSort(),
		"starred_at":     "starred, starred_at",
	})

	return r
}

func recentlyAddedSort() string {
	if conf.Server.RecentlyAddedByModTime {
		return "updated_at"
	}
	return "created_at"
}

func recentlyPlayedFilter(string, interface{}) Sqlizer {
	return Gt{"play_count": 0}
}

func hasRatingFilter(string, interface{}) Sqlizer {
	return Gt{"rating": 0}
}

func yearFilter(_ string, value interface{}) Sqlizer {
	return Or{
		And{
			Gt{"min_year": 0},
			LtOrEq{"min_year": value},
			GtOrEq{"max_year": value},
		},
		Eq{"max_year": value},
	}
}

func artistFilter(_ string, value interface{}) Sqlizer {
	return Like{"participant_ids": fmt.Sprintf(`%%"%s"%%`, value)}
}

func (r *albumRepository) CountAll(options ...model.QueryOptions) (int64, error) {
	sql := r.newSelect()
	sql = r.withAnnotation(sql, "album.id")
	// BFR WithParticipants (for filter)?
	return r.count(sql, options...)
}

func (r *albumRepository) Exists(id string) (bool, error) {
	return r.exists(Eq{"album.id": id})
}

func (r *albumRepository) Put(al *model.Album) error {
	al.ImportedAt = time.Now()
	id, err := r.put(al.ID, &dbAlbum{Album: al})
	if err != nil {
		return err
	}
	al.ID = id
	// Only update participations and tags if there are any. Not the best place to put this,
	// but updating external metadata does not provide these fields.
	// TODO Move external metadata to a separated table
	if len(al.Participations) > 0 {
		err = r.updateParticipations(al.ID, al.Participations)
		if err != nil {
			return err
		}
	}
	return err
}

func (r *albumRepository) selectAlbum(options ...model.QueryOptions) SelectBuilder {
	sql := r.newSelect(options...).Columns("album.*")
	return r.withAnnotation(sql, "album.id")
}

func (r *albumRepository) Get(id string) (*model.Album, error) {
	res, err := r.GetAll(model.QueryOptions{Filters: Eq{"album.id": id}})
	if err != nil {
		return nil, err
	}
	if len(res) == 0 {
		return nil, model.ErrNotFound
	}
	return &res[0], nil
}

func (r *albumRepository) GetAll(options ...model.QueryOptions) (model.Albums, error) {
	sq := r.selectAlbum(options...)
	var res dbAlbums
	err := r.queryAll(sq, &res)
	if err != nil {
		return nil, err
	}
	err = r.loadTags(&res)
	if err != nil {
		return nil, err
	}
	err = r.loadParticipations(&res)
	if err != nil {
		return nil, err
	}
	return res.toModels(), err
}

// Touch flags an album as being scanned by the scanner, but not necessarily updated.
// This is used for when missing tracks are detected for an album during scan.
func (r *albumRepository) Touch(ids ...string) error {
	if len(ids) == 0 {
		return nil
	}
	upd := Update(r.tableName).Set("imported_at", timeToSQL(time.Now())).Where(Eq{"id": ids})
	c, err := r.executeSQL(upd)
	if err == nil {
		log.Debug(r.ctx, "Touching albums", "ids", ids, "updated", c == 1)
	}
	return err
}

// GetTouchedAlbums returns a list of albums that were touched by the scanner for a given library, in the
// current library scan.
func (r *albumRepository) GetTouchedAlbums(libID int) (model.Albums, error) {
	sel := r.selectAlbum().
		Join("library on library.id = album.library_id").
		Where(And{
			Eq{"library.id": libID},
			ConcatExpr("album.imported_at > library.last_scan_at"),
		})
	var res dbAlbums
	err := r.queryAll(sel, &res)
	if err != nil {
		return nil, err
	}
	err = r.loadTags(&res)
	if err != nil {
		return nil, err
	}
	err = r.loadParticipations(&res)
	return res.toModels(), err
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
	var res dbAlbums
	err := r.doSearch(q, offset, size, &res, "name")
	if err != nil {
		return nil, err
	}
	err = r.loadTags(&res)
	if err != nil {
		return nil, err
	}
	err = r.loadParticipations(&res)
	return res.toModels(), err
}

func (r *albumRepository) Count(options ...rest.QueryOptions) (int64, error) {
	return r.CountAll(r.parseRestOptions(r.ctx, options...))
}

func (r *albumRepository) Read(id string) (interface{}, error) {
	return r.Get(id)
}

func (r *albumRepository) ReadAll(options ...rest.QueryOptions) (interface{}, error) {
	return r.GetAll(r.parseRestOptions(r.ctx, options...))
}

func (r *albumRepository) EntityName() string {
	return "album"
}

func (r *albumRepository) NewInstance() interface{} {
	return &model.Album{}
}

var _ model.AlbumRepository = (*albumRepository)(nil)
var _ model.ResourceRepository = (*albumRepository)(nil)
