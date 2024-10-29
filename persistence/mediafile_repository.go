package persistence

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	. "github.com/Masterminds/squirrel"
	"github.com/deluan/rest"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/utils/slice"
	"github.com/pocketbase/dbx"
)

type mediaFileRepository struct {
	sqlRepository
}

type dbMediaFile struct {
	*model.MediaFile `structs:",flatten"`
	ParticipantIDs   string `structs:"-" json:"-"`
	TagIds           string `structs:"-" json:"-"`
	parsedTagIDs     []string
}

func (m *dbMediaFile) PostScan() error {
	m.MediaFile.Participations = parseParticipations(m.ParticipantIDs)
	if m.TagIds == "" {
		return nil
	}
	err := json.Unmarshal([]byte(m.TagIds), &m.parsedTagIDs)
	if err != nil {
		return fmt.Errorf("error parsing media_file tags: %w", err)
	}
	return nil
}

func (m *dbMediaFile) PostMapArgs(args map[string]any) error {
	fullText := []string{m.Title, m.Album, m.Artist, m.AlbumArtist,
		m.SortTitle, m.SortAlbumName, m.SortArtistName, m.SortAlbumArtistName, m.DiscSubtitle}
	fullText = append(fullText, m.MediaFile.Participations.AllNames()...)
	args["full_text"] = formatFullText(fullText...)
	args["tag_ids"] = buildTagIDs(m.MediaFile.Tags)
	args["participant_ids"] = buildParticipantIDs(m.MediaFile.Participations)
	delete(args, "tags")
	delete(args, "participations")
	return nil
}

func (m *dbMediaFile) tagIDs() []string {
	return m.parsedTagIDs
}

type dbMediaFiles []dbMediaFile

func (m dbMediaFiles) toModels() model.MediaFiles {
	return slice.Map(m, func(mf dbMediaFile) model.MediaFile { return *mf.MediaFile })
}

func (m dbMediaFiles) tagIDs() []string {
	var ids []string
	for _, mf := range m {
		ids = append(ids, mf.parsedTagIDs...)
	}
	return slice.Unique(ids)
}

func (m dbMediaFiles) setTags(tagMap map[string]model.Tag) {
	for i, mf := range m {
		tags := model.Tags{}
		for _, id := range mf.parsedTagIDs {
			if tag, ok := tagMap[id]; ok {
				tags[tag.TagName] = append(tags[tag.TagName], tag.TagValue)
			}
		}
		m[i].MediaFile.Tags = tags
		m[i].MediaFile.Genre, m[i].MediaFile.Genres = tags.ToGenres()
	}
}

func (m dbMediaFiles) getParticipantIDs() []string {
	var ids []string
	for _, mf := range m {
		ids = append(ids, mf.Participations.AllIDs()...)
	}
	return slice.Unique(ids)
}

func (m dbMediaFiles) setParticipations(participantMap map[string]string) {
	for i, mf := range m {
		for role, artists := range mf.MediaFile.Participations {
			for j, artist := range artists {
				if name, ok := participantMap[artist.ID]; ok {
					m[i].MediaFile.Participations[role][j].Name = name
				}
			}
		}
	}
}

func NewMediaFileRepository(ctx context.Context, db dbx.Builder) model.MediaFileRepository {
	r := &mediaFileRepository{}
	r.ctx = ctx
	r.db = db
	r.tableName = "media_file"
	r.registerModel(&model.MediaFile{}, map[string]filterFunc{
		"id":       idFilter(r.tableName),
		"title":    fullTextFilter(r.tableName),
		"starred":  booleanFilter,
		"genre_id": tagIDFilter,
	})
	r.setSortMappings(map[string]string{
		"title":        "order_title",
		"artist":       "order_artist_name, order_album_name, release_date, disc_number, track_number",
		"album_artist": "order_album_artist_name, order_album_name, release_date, disc_number, track_number",
		"album":        "order_album_name, release_date, disc_number, track_number, order_artist_name, title",
		"random":       "random",
		"created_at":   "media_file.created_at",
		"starred_at":   "starred, starred_at",
	})
	return r
}

func (r *mediaFileRepository) CountAll(options ...model.QueryOptions) (int64, error) {
	query := r.newSelect()
	query = r.withAnnotation(query, "media_file.id")
	// BFR WithParticipants (for filtering by name)?
	return r.count(query, options...)
}

func (r *mediaFileRepository) Exists(id string) (bool, error) {
	return r.exists(Eq{"media_file.id": id})
}

func (r *mediaFileRepository) Put(m *model.MediaFile) error {
	m.CreatedAt = time.Now()
	id, err := r.putByMatch(Eq{"path": m.Path, "library_id": m.LibraryID}, m.ID, &dbMediaFile{MediaFile: m})
	if err != nil {
		return err
	}
	m.ID = id
	return r.updateParticipations(m.ID, m.Participations)
}

func (r *mediaFileRepository) selectMediaFile(options ...model.QueryOptions) SelectBuilder {
	sql := r.newSelect(options...).Columns("media_file.*")
	sql = r.withAnnotation(sql, "media_file.id")
	return r.withBookmark(sql, "media_file.id")
}

func (r *mediaFileRepository) Get(id string) (*model.MediaFile, error) {
	res, err := r.GetAll(model.QueryOptions{Filters: Eq{"media_file.id": id}})
	if err != nil {
		return nil, err
	}
	if len(res) == 0 {
		return nil, model.ErrNotFound
	}
	return &res[0], nil
}

func (r *mediaFileRepository) GetAll(options ...model.QueryOptions) (model.MediaFiles, error) {
	sq := r.selectMediaFile(options...)
	var res dbMediaFiles
	err := r.queryAll(sq, &res, options...)
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
	return res.toModels(), nil
}

func (r *mediaFileRepository) FindByPaths(paths []string) (model.MediaFiles, error) {
	sel := r.newSelect().Columns("*").Where(Eq{"path collate nocase": paths})
	var res model.MediaFiles
	if err := r.queryAll(sel, &res); err != nil {
		return nil, err
	}
	return res, nil
}

func cleanPath(path string) string {
	path = filepath.Clean(path)
	if !strings.HasSuffix(path, string(os.PathSeparator)) {
		path += string(os.PathSeparator)
	}
	return path
}

func pathStartsWith(path string) Eq {
	substr := fmt.Sprintf("substr(path, 1, %d)", utf8.RuneCountInString(path))
	return Eq{substr: path}
}

// FindAllByPath only return mediafiles that are direct children of requested path
func (r *mediaFileRepository) FindAllByPath(path string) (model.MediaFiles, error) {
	// Query by path based on https://stackoverflow.com/a/13911906/653632
	path = cleanPath(path)
	pathLen := utf8.RuneCountInString(path)
	sel0 := r.newSelect().Columns("media_file.*", fmt.Sprintf("substr(path, %d) AS item", pathLen+2)).
		Where(pathStartsWith(path))
	sel := r.newSelect().Columns("*", "item NOT GLOB '*"+string(os.PathSeparator)+"*' AS isLast").
		Where(Eq{"isLast": 1}).FromSelect(sel0, "sel0")

	res := dbMediaFiles{}
	err := r.queryAll(sel, &res)
	return res.toModels(), err
}

// FindPathsRecursively returns a list of all subfolders of basePath, recursively
func (r *mediaFileRepository) FindPathsRecursively(basePath string) ([]string, error) {
	path := cleanPath(basePath)
	// Query based on https://stackoverflow.com/a/38330814/653632
	sel := r.newSelect().Columns(fmt.Sprintf("distinct rtrim(path, replace(path, '%s', ''))", string(os.PathSeparator))).
		Where(pathStartsWith(path))
	var res []string
	err := r.queryAllSlice(sel, &res)
	return res, err
}

func (r *mediaFileRepository) deleteNotInPath(basePath string) error {
	path := cleanPath(basePath)
	sel := Delete(r.tableName).Where(NotEq(pathStartsWith(path)))
	c, err := r.executeSQL(sel)
	if err == nil {
		if c > 0 {
			log.Debug(r.ctx, "Deleted dangling tracks", "totalDeleted", c)
		}
	}
	return err
}

func (r *mediaFileRepository) Delete(id string) error {
	return r.delete(Eq{"id": id})
}

// DeleteByPath delete from the DB all mediafiles that are direct children of path
func (r *mediaFileRepository) DeleteByPath(basePath string) (int64, error) {
	path := cleanPath(basePath)
	pathLen := utf8.RuneCountInString(path)
	del := Delete(r.tableName).
		Where(And{pathStartsWith(path),
			Eq{fmt.Sprintf("substr(path, %d) glob '*%s*'", pathLen+2, string(os.PathSeparator)): 0}})
	log.Debug(r.ctx, "Deleting mediafiles by path", "path", path)
	return r.executeSQL(del)
}

func (r *mediaFileRepository) MarkMissing(missing bool, mfs ...model.MediaFile) error {
	ids := slice.SeqFunc(mfs, func(m model.MediaFile) string { return m.ID })
	for chunk := range slice.CollectChunks(ids, 200) {
		upd := Update(r.tableName).
			Set("missing", missing).
			Set("updated_at", timeToSQL(time.Now())).
			Where(Eq{"id": chunk})
		c, err := r.executeSQL(upd)
		if err != nil || c == 0 {
			log.Error(r.ctx, "Error setting mediafile missing flag", "ids", chunk, err)
			return err
		}
		log.Debug(r.ctx, "Marked missing mediafiles", "total", c, "ids", chunk)
	}
	return nil
}

func (r *mediaFileRepository) MarkMissingByFolder(missing bool, folderIDs ...string) error {
	for chunk := range slices.Chunk(folderIDs, 200) {
		upd := Update(r.tableName).
			Set("missing", missing).
			Set("updated_at", timeToSQL(time.Now())).
			Where(And{
				Eq{"folder_id": chunk},
				Eq{"missing": !missing},
			})
		c, err := r.executeSQL(upd)
		if err != nil {
			log.Error(r.ctx, "Error setting mediafile missing flag", "folderIDs", chunk, err)
			return err
		}
		log.Debug(r.ctx, "Marked missing mediafiles from missing folders", "total", c, "folders", chunk)
	}
	return nil
}

// GetMissingAndMatching returns all mediafiles that are missing and their potential matches (comparing PIDs)
// that were added/updated after the last scan started
func (r *mediaFileRepository) GetMissingAndMatching(libId int, pagination ...model.QueryOptions) (model.MediaFiles, error) {
	subQ := r.newSelect().Columns("pid").
		Where(And{
			Eq{"media_file.missing": true},
			Eq{"library_id": libId},
		})
	subQText, subQArgs, err := subQ.PlaceholderFormat(Question).ToSql()
	if err != nil {
		return nil, err
	}
	sel := r.selectMediaFile(pagination...).
		Where("pid in ("+subQText+")", subQArgs...).
		Where(Or{
			Eq{"missing": true},
			ConcatExpr("media_file.created_at > library.last_scan_started_at"),
		}).
		Join("library on media_file.library_id = library.id").
		OrderBy("pid")
	var res dbMediaFiles
	err = r.queryAll(sel, &res)
	if err != nil {
		return nil, err
	}
	err = r.loadTags(&res)
	if err != nil {
		return nil, err
	}
	//err = r.loadParticipations(&res) BFR Needed?
	return res.toModels(), nil
}

func (r *mediaFileRepository) removeNonAlbumArtistIds() error {
	upd := Update(r.tableName).Set("artist_id", "").Where(notExists("artist", ConcatExpr("id = artist_id")))
	log.Debug(r.ctx, "Removing non-album artist_ids")
	_, err := r.executeSQL(upd)
	return err
}

func (r *mediaFileRepository) Search(q string, offset int, size int) (model.MediaFiles, error) {
	results := dbMediaFiles{}
	err := r.doSearch(q, offset, size, &results, "title")
	if err != nil {
		return nil, err
	}
	err = r.loadTags(&results)
	if err != nil {
		return nil, err
	}
	err = r.loadParticipations(&results)
	return results.toModels(), err
}

func (r *mediaFileRepository) Count(options ...rest.QueryOptions) (int64, error) {
	return r.CountAll(r.parseRestOptions(r.ctx, options...))
}

func (r *mediaFileRepository) Read(id string) (interface{}, error) {
	return r.Get(id)
}

func (r *mediaFileRepository) ReadAll(options ...rest.QueryOptions) (interface{}, error) {
	return r.GetAll(r.parseRestOptions(r.ctx, options...))
}

func (r *mediaFileRepository) EntityName() string {
	return "mediafile"
}

func (r *mediaFileRepository) NewInstance() interface{} {
	return &model.MediaFile{}
}

var _ model.MediaFileRepository = (*mediaFileRepository)(nil)
var _ model.ResourceRepository = (*mediaFileRepository)(nil)
