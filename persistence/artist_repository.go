package persistence

import (
	"cmp"
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"slices"
	"strings"
	"time"

	. "github.com/Masterminds/squirrel"
	"github.com/deluan/rest"
	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/utils"
	. "github.com/navidrome/navidrome/utils/gg"
	"github.com/navidrome/navidrome/utils/slice"
	"github.com/pocketbase/dbx"
)

type artistRepository struct {
	sqlRepository
	indexGroups utils.IndexGroups
}

type dbArtist struct {
	*model.Artist  `structs:",flatten"`
	SimilarArtists string `structs:"-" json:"similarArtists"`
	Stats          string `structs:"-" json:"-"`
}

func (a *dbArtist) PostScan() error {
	var stats map[string]map[string]int64
	if err := json.Unmarshal([]byte(a.Stats), &stats); err != nil {
		return fmt.Errorf("parsing artist stats from db: %w", err)
	}
	a.Artist.Stats = make(map[model.Role]model.ArtistStats)
	for key, c := range stats {
		if key == "total" {
			a.Artist.Size = c["s"]
			a.Artist.SongCount = int(c["m"])
			a.Artist.AlbumCount = int(c["a"])
		}
		role := model.RoleFromString(key)
		if role == model.RoleInvalid {
			continue
		}
		a.Artist.Stats[role] = model.ArtistStats{
			SongCount:  int(c["m"]),
			AlbumCount: int(c["a"]),
			Size:       c["s"],
		}
	}
	a.Artist.SimilarArtists = nil
	if a.SimilarArtists == "" {
		return nil
	}
	// BFR: Save similar artists as JSONB in the DB
	for _, s := range strings.Split(a.SimilarArtists, ";") {
		fields := strings.Split(s, ":")
		if len(fields) != 2 {
			continue
		}
		name, _ := url.QueryUnescape(fields[1])
		a.Artist.SimilarArtists = append(a.Artist.SimilarArtists, model.Artist{
			ID:   fields[0],
			Name: name,
		})
	}
	return nil
}
func (a *dbArtist) PostMapArgs(m map[string]any) error {
	var sa []string
	for _, s := range a.Artist.SimilarArtists {
		sa = append(sa, fmt.Sprintf("%s:%s", s.ID, url.QueryEscape(s.Name)))
	}
	m["similar_artists"] = strings.Join(sa, ";")
	m["full_text"] = formatFullText(a.Name, a.SortArtistName)

	// Do not override the sort_artist_name and mbz_artist_id fields if they are empty
	// BFR: Better way to handle this?
	if v, ok := m["sort_artist_name"]; !ok || v.(string) == "" {
		delete(m, "sort_artist_name")
	}
	if v, ok := m["mbz_artist_id"]; !ok || v.(string) == "" {
		delete(m, "mbz_artist_id")
	}
	return nil
}

type dbArtists []dbArtist

func (dba dbArtists) toModels() model.Artists {
	res := make(model.Artists, len(dba))
	for i := range dba {
		res[i] = *dba[i].Artist
	}
	return res
}

func NewArtistRepository(ctx context.Context, db dbx.Builder) model.ArtistRepository {
	r := &artistRepository{}
	r.ctx = ctx
	r.db = db
	r.indexGroups = utils.ParseIndexGroups(conf.Server.IndexGroups)
	r.tableName = "artist" // To be used by the idFilter below
	r.registerModel(&model.Artist{}, map[string]filterFunc{
		"id":      idFilter(r.tableName),
		"name":    fullTextFilter(r.tableName),
		"starred": booleanFilter,
		"role":    roleFilter,
	})
	r.setSortMappings(map[string]string{
		"name":       "order_artist_name",
		"starred_at": "starred, starred_at",
		// BFR: Can be dynamic (by role) if we allow functions when sorting
		"song_count":  "stats->>'total'->>'m'",
		"album_count": "stats->>'total'->>'a'",
		"size":        "stats->>'total'->>'s'",
	})
	return r
}

func roleFilter(_ string, role any) Sqlizer {
	return NotEq{fmt.Sprintf("stats ->> '$.%v'", role): nil}
}

func (r *artistRepository) selectArtist(options ...model.QueryOptions) SelectBuilder {
	query := r.newSelect(options...).Columns("artist.*")
	query = r.withAnnotation(query, "artist.id")
	// BFR How to handle counts and sizes (per role)?
	return query
}

func (r *artistRepository) CountAll(options ...model.QueryOptions) (int64, error) {
	query := r.newSelect()
	query = r.withAnnotation(query, "artist.id")
	return r.count(query, options...)
}

func (r *artistRepository) Exists(id string) (bool, error) {
	return r.exists(Eq{"artist.id": id})
}

func (r *artistRepository) Put(a *model.Artist, colsToUpdate ...string) error {
	dba := &dbArtist{Artist: a}
	dba.CreatedAt = P(time.Now())
	dba.UpdatedAt = dba.CreatedAt
	_, err := r.put(dba.ID, dba, colsToUpdate...)
	return err
}

func (r *artistRepository) Get(id string) (*model.Artist, error) {
	sel := r.selectArtist().Where(Eq{"artist.id": id})
	var dba dbArtists
	if err := r.queryAll(sel, &dba); err != nil {
		return nil, err
	}
	if len(dba) == 0 {
		return nil, model.ErrNotFound
	}
	res := dba.toModels()
	return &res[0], nil
}

func (r *artistRepository) GetAll(options ...model.QueryOptions) (model.Artists, error) {
	sel := r.selectArtist(options...)
	var dba dbArtists
	err := r.queryAll(sel, &dba)
	if err != nil {
		return nil, err
	}
	res := dba.toModels()
	return res, err
}

func (r *artistRepository) getIndexKey(a model.Artist) string {
	source := a.OrderArtistName
	if conf.Server.PreferSortTags {
		source = cmp.Or(a.SortArtistName, a.OrderArtistName)
	}
	name := strings.ToLower(source)
	for k, v := range r.indexGroups {
		if strings.HasPrefix(name, strings.ToLower(k)) {
			return v
		}
	}
	return "#"
}

// TODO Cache the index (recalculate when there are changes to the DB)
func (r *artistRepository) GetIndex() (model.ArtistIndexes, error) {
	artists, err := r.GetAll(model.QueryOptions{Sort: "name"})
	if err != nil {
		return nil, err
	}
	var result model.ArtistIndexes
	for k, v := range slice.Group(artists, r.getIndexKey) {
		result = append(result, model.ArtistIndex{ID: k, Artists: v})
	}
	slices.SortFunc(result, func(a, b model.ArtistIndex) int {
		return cmp.Compare(a.ID, b.ID)
	})
	return result, nil
}

func (r *artistRepository) purgeEmpty() error {
	del := Delete(r.tableName).Where("id not in (select artist_id from album_artists)")
	c, err := r.executeSQL(del)
	if err != nil {
		return fmt.Errorf("purging empty artists: %w", err)
	}
	if c > 0 {
		log.Debug(r.ctx, "Purged empty artists", "totalDeleted", c)
	}
	return nil
}

// RefreshAnnotations updates the play count and last play date annotations for all artists, based
// on the media files associated with them.
func (r *artistRepository) RefreshAnnotations() (int64, error) {
	query := rawSQL(`
with play_counts as (
    select user_id, atom as artist_id, sum(play_count) as total_play_count, max(play_date) as last_play_date
    from media_file
    join annotation on item_id = media_file.id
    left join json_tree(participant_ids, '$.artist') as jt
    where atom is not null
    group by user_id, atom
)
insert or replace into annotation
(item_id, item_type, play_count, play_date, user_id)
select artist_id, 'artist', total_play_count, last_play_date, user_id
from play_counts;
`)
	return r.executeSQL(query)
}

// RefreshStats updates the stats field for all artists, based on the media files associated with them.
// BFR Maybe filter by "touched" artists?
func (r *artistRepository) RefreshStats() (int64, error) {
	// First get all counters, one query groups by artist/role, and another with totals per artist.
	// Union both queries and group by artist to get a single row of counters per artist/role.
	// Then format the counters in a JSON object, one key for each role.
	// Finally update the artist table with the new counters
	// In all queries, atom is the artist ID and path is the role (or "total" for the totals)
	query := rawSQL(`
-- CTE to get counters for each artist, grouped by role
with artist_role_counters as (
   -- Get counters for each artist, grouped by role 
   -- (remove the index from the role: composer[0] => composer
    select atom,
           substr(
               replace(jt.path, '$.', ''),
               1,
               case when instr(replace(jt.path, '$.', ''), '[') > 0
                    then instr(replace(jt.path, '$.', ''), '[') - 1
                    else length(replace(jt.path, '$.', ''))
               end
           ) as path,
           count(distinct album_id) as album_count,
           count(mf.id) as count,
           sum(size) as size
    from media_file mf
             left join json_tree(participant_ids) jt
    where atom is not null
    group by atom, jt.path
),

-- CTE to get the totals for each artist
artist_total_counters as (
    select mfa.artist_id as atom,
           'total' as path,
           count(distinct mf.album) as album_count,
           count(distinct mf.id) as count,
           sum(mf.size) as size
    from (select distinct artist_id, media_file_id
          from main.media_file_artists) as mfa
             join main.media_file mf on mfa.media_file_id = mf.id
    group by mfa.artist_id
),

-- CTE to combine role and total counters
combined_counters as (
    select atom, path, album_count, count, size
    from artist_role_counters
    union
    select atom, path, album_count, count, size
    from artist_total_counters
),

-- CTE to format the counters in a JSON object
artist_counters as (
    select atom as id,
           json_group_object(
               replace(path, '"', ''),
               json_object('a', album_count, 'm', count, 's', size)
           ) as counters
    from combined_counters
    group by atom
)

-- Update the artist table with the new counters
update artist
set stats = coalesce((select counters from artist_counters where artist_counters.id = artist.id), '{}'),
    updated_at = current_timestamp
where id <> ''; -- always true, to avoid warnings`)
	return r.executeSQL(query)
}

func (r *artistRepository) Search(q string, offset int, size int) (model.Artists, error) {
	var dba dbArtists
	err := r.doSearch(q, offset, size, &dba, "name")
	if err != nil {
		return nil, err
	}
	return dba.toModels(), nil
}

func (r *artistRepository) Count(options ...rest.QueryOptions) (int64, error) {
	return r.CountAll(r.parseRestOptions(r.ctx, options...))
}

func (r *artistRepository) Read(id string) (interface{}, error) {
	return r.Get(id)
}

func (r *artistRepository) ReadAll(options ...rest.QueryOptions) (interface{}, error) {
	return r.GetAll(r.parseRestOptions(r.ctx, options...))
}

func (r *artistRepository) EntityName() string {
	return "artist"
}

func (r *artistRepository) NewInstance() interface{} {
	return &model.Artist{}
}

var _ model.ArtistRepository = (*artistRepository)(nil)
var _ model.ResourceRepository = (*artistRepository)(nil)
