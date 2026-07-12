package persistence

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"

	. "github.com/Masterminds/squirrel"
	"github.com/deluan/rest"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/pocketbase/dbx"
)

type playlistRepository struct {
	sqlRepository
}

type dbPlaylist struct {
	model.Playlist   `structs:",flatten"`
	Rules            sql.NullString `structs:"-"`
	PhysicalFolderID string         `db:"physical_folder_id" structs:"physical_folder_id"`
}

func (p *dbPlaylist) PostScan() error {
	if p.Rules.String != "" {
		return json.Unmarshal([]byte(p.Rules.String), &p.Playlist.Rules)
	}
	return nil
}

func (p dbPlaylist) PostMapArgs(args map[string]any) error {
	var err error
	if p.Playlist.IsSmartPlaylist() {
		args["rules"], err = json.Marshal(p.Playlist.Rules)
		if err != nil {
			return fmt.Errorf("invalid criteria expression: %w", err)
		}
		return nil
	}
	delete(args, "rules")
	return nil
}

func NewPlaylistRepository(ctx context.Context, db dbx.Builder) model.PlaylistRepository {
	r := &playlistRepository{}
	r.ctx = ctx
	r.db = db
	r.registerModel(&model.Playlist{}, map[string]filterFunc{
		"q":                  playlistFilter,
		"smart":              smartPlaylistFilter,
		"physical_folder_id": eqFilter,
	})
	r.setSortMappings(map[string]string{
		"owner_name": "owner_name",
	})
	return r
}

func playlistFilter(_ string, value any) Sqlizer {
	return Or{
		substringFilter("playlist.name", value),
		substringFilter("playlist.comment", value),
	}
}

func smartPlaylistFilter(string, any) Sqlizer {
	return Or{
		Eq{"rules": ""},
		Eq{"rules": nil},
	}
}

func (r *playlistRepository) userFilter() Sqlizer {
	user := loggedUser(r.ctx)
	if user.IsAdmin {
		return And{}
	}
	return Or{
		Eq{"public": true},
		Eq{"owner_id": user.ID},
	}
}

func (r *playlistRepository) CountAll(options ...model.QueryOptions) (int64, error) {
	sq := r.newSelect().Where(r.userFilter())
	return r.count(sq, options...)
}

func (r *playlistRepository) Exists(id string) (bool, error) {
	return r.exists(And{Eq{"id": id}, r.userFilter()})
}

func (r *playlistRepository) Delete(id string) error {
	return r.delete(And{Eq{"id": id}, r.userFilter()})
}

func (r *playlistRepository) Put(p *model.Playlist, cols ...string) error {
	pls := dbPlaylist{Playlist: *p}
	if len(cols) > 0 {
		if pls.ID == "" {
			return errors.New("playlist id is required for partial update")
		}
		_, err := r.put(pls.ID, pls, cols...)
		return err
	}
	if pls.ID == "" {
		pls.CreatedAt = time.Now()
	}
	pls.UpdatedAt = time.Now()

	id, err := r.put(pls.ID, pls)
	if err != nil {
		return err
	}
	p.ID = id

	if p.IsSmartPlaylist() {
		// Do not update tracks at this point, as it may take a long time and lock the DB, breaking the scan process
		return nil
	}
	// Only update tracks if they were specified
	if len(pls.Tracks) > 0 {
		return r.updateTracks(id, pls.Tracks)
	}
	return r.refreshCounters(&pls.Playlist)
}

func (r *playlistRepository) Get(id string) (*model.Playlist, error) {
	return r.findBy(And{Eq{"playlist.id": id}, r.userFilter()})
}

func (r *playlistRepository) GetWithTracks(id string, refreshSmartPlaylist, includeMissing bool) (*model.Playlist, error) {
	pls, err := r.Get(id)
	if err != nil {
		return nil, err
	}
	if refreshSmartPlaylist {
		r.refreshSmartPlaylist(pls)
	}
	tracks, err := r.loadTracks(id, model.QueryOptions{Sort: "id"}, true)
	if err != nil {
		log.Error(r.ctx, "Error loading playlist tracks ", "playlist", pls.Name, "id", pls.ID, err)
		return nil, err
	}
	pls.SetTracks(tracks)
	return pls, nil
}

func (r *playlistRepository) FindByPath(path string) (*model.Playlist, error) {
	return r.findBy(Eq{"path": path})
}

func (r *playlistRepository) findBy(sql Sqlizer) (*model.Playlist, error) {
	sel := r.selectPlaylist().Where(sql)
	var pls []dbPlaylist
	err := r.queryAll(sel, &pls)
	if err != nil {
		return nil, err
	}
	if len(pls) == 0 {
		return nil, model.ErrNotFound
	}

	return &pls[0].Playlist, nil
}

func (r *playlistRepository) GetAll(options ...model.QueryOptions) (model.Playlists, error) {
	sel := r.selectPlaylist(options...).Where(r.userFilter())
	var res []dbPlaylist
	err := r.queryAll(sel, &res)
	if err != nil {
		return nil, err
	}
	playlists := make(model.Playlists, len(res))
	for i, p := range res {
		playlists[i] = p.Playlist
	}
	return playlists, err
}

func (r *playlistRepository) GetPlaylists(itemID string) (model.Playlists, error) {
	sel := r.selectPlaylist(model.QueryOptions{Sort: "name"}).
		Join("playlist_tracks on playlist.id = playlist_tracks.playlist_id").
		Where(And{Eq{"playlist_tracks.media_file_id": itemID}, r.userFilter()})
	var res []dbPlaylist
	err := r.queryAll(sel, &res)
	if err != nil {
		if errors.Is(err, model.ErrNotFound) {
			return model.Playlists{}, nil
		}
		return nil, err
	}
	playlists := make(model.Playlists, len(res))
	for i, p := range res {
		playlists[i] = p.Playlist
	}
	return playlists, nil
}

func (r *playlistRepository) GetSyncPlaylists() (model.Playlists, error) {
	sel := r.selectPlaylist().Where(And{
		NotEq{"physical_folder_id": ""},
		NotEq{"physical_folder_id": nil},
	})
	var res []dbPlaylist
	err := r.queryAll(sel, &res)
	if err != nil {
		return nil, err
	}
	playlists := make(model.Playlists, len(res))
	for i, p := range res {
		playlists[i] = p.Playlist
	}
	return playlists, nil
}

func (r *playlistRepository) selectPlaylist(options ...model.QueryOptions) SelectBuilder {
	return r.newSelect(options...).Join("user on user.id = owner_id").
		Columns(r.tableName+".*", "user.user_name as owner_name")
}

func (r *playlistRepository) updateTracks(id string, tracks model.PlaylistTracks) error {
	refs := make([]model.PlaylistTrackRef, len(tracks))
	for i, t := range tracks {
		itemType := t.ItemType
		if itemType == "" {
			itemType = model.PlaylistTrackSong
		}
		refs[i] = model.PlaylistTrackRef{ID: t.MediaFileID, ItemType: itemType}
	}
	return r.updatePlaylist(id, refs)
}

func (r *playlistRepository) updatePlaylist(playlistId string, refs []model.PlaylistTrackRef) error {
	// Remove old tracks
	del := Delete("playlist_tracks").Where(Eq{"playlist_id": playlistId})
	_, err := r.executeSQL(del)
	if err != nil {
		return err
	}

	return r.addTracks(playlistId, 1, refs)
}

func (r *playlistRepository) addTracks(playlistId string, startingPos int, refs []model.PlaylistTrackRef) error {
	// Break the track list in chunks to avoid hitting SQLITE_MAX_VARIABLE_NUMBER limit
	// Add new tracks, chunk by chunk
	pos := startingPos
	for chunk := range slices.Chunk(refs, 200) {
		ins := Insert("playlist_tracks").Columns("playlist_id", "media_file_id", "id", "item_type")
		for _, ref := range chunk {
			itemType := ref.ItemType
			if itemType == "" {
				itemType = model.PlaylistTrackSong
			}
			ins = ins.Values(playlistId, ref.ID, pos, string(itemType))
			pos++
		}
		_, err := r.executeSQL(ins)
		if err != nil {
			return err
		}
	}

	return r.refreshCounters(&model.Playlist{ID: playlistId})
}

// refreshCounters updates total playlist duration, size and count, summing
// across both song (media_file) and downloaded podcast episode tracks.
func (r *playlistRepository) refreshCounters(pls *model.Playlist) error {
	songStats := Select(
		"coalesce(sum(duration), 0) as duration",
		"coalesce(sum(size), 0) as size",
		"count(*) as count",
	).
		From("media_file").
		Join("playlist_tracks f on f.media_file_id = media_file.id").
		Where(And{Eq{"playlist_id": pls.ID}, Eq{"f.item_type": string(model.PlaylistTrackSong)}})
	var songRes struct{ Duration, Size, Count float32 }
	if err := r.queryOne(songStats, &songRes); err != nil {
		return err
	}

	episodeStats := Select(
		"coalesce(sum(duration), 0) as duration",
		"coalesce(sum(size), 0) as size",
		"count(*) as count",
	).
		From("podcast_episode").
		Join("playlist_tracks f on f.media_file_id = podcast_episode.id").
		Where(And{Eq{"playlist_id": pls.ID}, Eq{"f.item_type": string(model.PlaylistTrackPodcastEpisode)}})
	var episodeRes struct{ Duration, Size, Count float32 }
	if err := r.queryOne(episodeStats, &episodeRes); err != nil {
		return err
	}

	duration := songRes.Duration + episodeRes.Duration
	size := songRes.Size + episodeRes.Size
	count := songRes.Count + episodeRes.Count

	// Update playlist's total duration, size and count
	upd := Update("playlist").
		Set("duration", duration).
		Set("size", size).
		Set("song_count", count).
		Set("updated_at", time.Now()).
		Where(Eq{"id": pls.ID})
	_, err := r.executeSQL(upd)
	if err != nil {
		return err
	}
	pls.SongCount = int(count)
	pls.Duration = duration
	pls.Size = int64(size)
	return nil
}

// dbPlaylistEpisodeTrack scans a playlist track backed by a downloaded
// podcast episode. Unlike dbPlaylistTrack (which relies on f.* and
// playlist_tracks.* both having an "id" column and a PostScan fixup), this
// aliases the position column explicitly to sidestep that ambiguity, since
// there's no existing test coverage to lean on for the collision-tolerant
// approach with a new struct shape.
type dbPlaylistEpisodeTrack struct {
	model.PodcastEpisode `structs:",flatten"`
	ChannelTitle         string `db:"channel_title"`
	Position             int    `db:"position"`
}

func (t dbPlaylistEpisodeTrack) toPlaylistTrack(playlistID string) model.PlaylistTrack {
	ep := t.PodcastEpisode
	return model.PlaylistTrack{
		ID:          strconv.Itoa(t.Position),
		MediaFileID: ep.ID,
		PlaylistID:  playlistID,
		ItemType:    model.PlaylistTrackPodcastEpisode,
		MediaFile: model.MediaFile{
			ID:          ep.ID,
			Title:       ep.Title,
			OrderTitle:  ep.Title,
			Duration:    ep.Duration,
			Size:        ep.Size,
			Suffix:      ep.Suffix,
			BitRate:     ep.BitRate,
			Album:       t.ChannelTitle,
			Artist:      t.ChannelTitle,
			AlbumArtist: t.ChannelTitle,
			// Order* fields feed sortPlaylistTracks' comparators, which are
			// shared with song tracks.
			OrderAlbumName:       t.ChannelTitle,
			OrderArtistName:      t.ChannelTitle,
			OrderAlbumArtistName: t.ChannelTitle,
			// Path is the episode's full absolute on-disk path (not
			// library-relative - LibraryPath is left empty), so generic
			// MediaFile.AbsolutePath()/M3U export code works unmodified for
			// downloaded episode tracks.
			Path:      ep.AbsolutePath(),
			CreatedAt: ep.CreatedAt,
			UpdatedAt: ep.UpdatedAt,
		},
	}
}

// loadTracks loads a playlist's tracks, which may reference songs
// (media_file) and/or downloaded podcast episodes. The two are fetched with
// separate queries - podcast episodes have no library/annotation/missing
// concept, so folding them into the existing song query's joins isn't
// possible - then merged and sorted in Go via sortPlaylistTracks, using the
// same field mappings playlistTrackRepository previously pushed to SQL via
// setSortMappings, so existing sort-by-column behavior (e.g. "album", which
// orders by disc/track number within it) keeps working across the merged
// result.
func (r *playlistRepository) loadTracks(playlistID string, options model.QueryOptions, excludeMissingSongs bool) (model.PlaylistTracks, error) {
	userID := loggedUser(r.ctx).ID
	songSel := Select(
		"coalesce(starred, 0) as starred",
		"starred_at",
		"coalesce(play_count, 0) as play_count",
		"play_date",
		"coalesce(rating, 0) as rating",
		"rated_at",
		"f.*",
		"playlist_tracks.*",
		"library.path as library_path",
		"library.name as library_name",
	).From("playlist_tracks").
		LeftJoin("annotation on (" +
			"annotation.item_id = media_file_id" +
			" AND annotation.item_type = 'media_file'" +
			" AND annotation.user_id = '" + userID + "')").
		Join("media_file f on f.id = media_file_id").
		Join("library on f.library_id = library.id").
		Where(And{Eq{"playlist_id": playlistID}, Eq{"playlist_tracks.item_type": string(model.PlaylistTrackSong)}})
	songSel = r.applyLibraryFilter(songSel, "f")
	if excludeMissingSongs {
		songSel = songSel.Where(Eq{"f.missing": false})
	}
	songRows := dbPlaylistTracks{}
	if err := r.queryAll(songSel, &songRows); err != nil {
		return nil, err
	}

	episodeSel := Select("pe.*", "pc.title as channel_title", "playlist_tracks.id as position").
		From("playlist_tracks").
		Join("podcast_episode pe on pe.id = playlist_tracks.media_file_id").
		Join("podcast_channel pc on pc.id = pe.channel_id").
		Where(And{Eq{"playlist_tracks.playlist_id": playlistID}, Eq{"playlist_tracks.item_type": string(model.PlaylistTrackPodcastEpisode)}})
	var episodeRows []dbPlaylistEpisodeTrack
	if err := r.queryAll(episodeSel, &episodeRows); err != nil {
		return nil, err
	}

	tracks := songRows.toModels()
	for _, row := range episodeRows {
		tracks = append(tracks, row.toPlaylistTrack(playlistID))
	}

	sortPlaylistTracks(tracks, options.Sort, options.Order)

	if options.Offset > 0 {
		if options.Offset >= len(tracks) {
			return model.PlaylistTracks{}, nil
		}
		tracks = tracks[options.Offset:]
	}
	if options.Max > 0 && options.Max < len(tracks) {
		tracks = tracks[:options.Max]
	}

	return tracks, nil
}

// sortPlaylistTracks sorts a merged song+episode track list in place,
// mirroring the field mappings playlistTrackRepository previously handed to
// SQL via setSortMappings (id/artist/album_artist/album/title/duration/
// year/bpm/channels). Applied in Go, rather than pushed to SQL separately
// per branch, since after loadTracks both songs and episodes are already
// represented as comparable model.PlaylistTrack/MediaFile values.
func sortPlaylistTracks(tracks model.PlaylistTracks, sortField, order string) {
	less := playlistTrackLess(sortField)
	desc := strings.EqualFold(order, "desc")
	sort.SliceStable(tracks, func(i, j int) bool {
		if desc {
			return less(tracks[j], tracks[i])
		}
		return less(tracks[i], tracks[j])
	})
}

func playlistTrackLess(sortField string) func(a, b model.PlaylistTrack) bool {
	switch sortField {
	case "artist":
		return func(a, b model.PlaylistTrack) bool { return a.OrderArtistName < b.OrderArtistName }
	case "album_artist":
		return func(a, b model.PlaylistTrack) bool { return a.OrderAlbumArtistName < b.OrderAlbumArtistName }
	case "album":
		return func(a, b model.PlaylistTrack) bool {
			if a.OrderAlbumName != b.OrderAlbumName {
				return a.OrderAlbumName < b.OrderAlbumName
			}
			if a.AlbumID != b.AlbumID {
				return a.AlbumID < b.AlbumID
			}
			if a.DiscNumber != b.DiscNumber {
				return a.DiscNumber < b.DiscNumber
			}
			if a.TrackNumber != b.TrackNumber {
				return a.TrackNumber < b.TrackNumber
			}
			if a.OrderArtistName != b.OrderArtistName {
				return a.OrderArtistName < b.OrderArtistName
			}
			return a.Title < b.Title
		}
	case "title":
		return func(a, b model.PlaylistTrack) bool { return a.OrderTitle < b.OrderTitle }
	case "duration":
		return func(a, b model.PlaylistTrack) bool { return a.Duration < b.Duration }
	case "year":
		return func(a, b model.PlaylistTrack) bool { return a.Year < b.Year }
	case "bpm":
		return func(a, b model.PlaylistTrack) bool { return a.BPM < b.BPM }
	case "channels":
		return func(a, b model.PlaylistTrack) bool { return a.Channels < b.Channels }
	default: // "id" (position) and anything unrecognized
		return func(a, b model.PlaylistTrack) bool {
			pa, _ := strconv.Atoi(a.ID)
			pb, _ := strconv.Atoi(b.ID)
			return pa < pb
		}
	}
}

func (r *playlistRepository) Count(options ...rest.QueryOptions) (int64, error) {
	return r.CountAll(r.parseRestOptions(r.ctx, options...))
}

func (r *playlistRepository) Read(id string) (any, error) {
	return r.Get(id)
}

func (r *playlistRepository) ReadAll(options ...rest.QueryOptions) (any, error) {
	return r.GetAll(r.parseRestOptions(r.ctx, options...))
}

func (r *playlistRepository) EntityName() string {
	return "playlist"
}

func (r *playlistRepository) NewInstance() any {
	return &model.Playlist{}
}

func (r *playlistRepository) Save(entity any) (string, error) {
	pls := entity.(*model.Playlist)
	pls.ID = "" // Force new creation
	err := r.Put(pls)
	if err != nil {
		return "", err
	}
	return pls.ID, err
}

func (r *playlistRepository) Update(id string, entity any, cols ...string) error {
	pls := dbPlaylist{Playlist: *entity.(*model.Playlist)}
	pls.ID = id
	pls.UpdatedAt = time.Now()
	_, err := r.put(id, pls, append(cols, "updatedAt")...)
	if errors.Is(err, model.ErrNotFound) {
		return rest.ErrNotFound
	}
	return err
}

// removeOrphans cleans up playlist tracks whose underlying song no longer
// exists (e.g. removed from the library by a scan). Scoped to song tracks
// only - podcast episode tracks are cleaned up directly wherever an
// episode's download is deleted (see core/podcasts), since that's a
// deliberate action rather than something a library scan would detect.
func (r *playlistRepository) removeOrphans() error {
	sel := Select("playlist_tracks.playlist_id as id", "p.name").From("playlist_tracks").
		Join("playlist p on playlist_tracks.playlist_id = p.id").
		LeftJoin("media_file mf on playlist_tracks.media_file_id = mf.id").
		Where(And{Eq{"mf.id": nil}, Eq{"playlist_tracks.item_type": string(model.PlaylistTrackSong)}}).
		GroupBy("playlist_tracks.playlist_id")

	var pls []struct{ Id, Name string }
	err := r.queryAll(sel, &pls)
	if err != nil {
		return fmt.Errorf("fetching playlists with orphan tracks: %w", err)
	}

	for _, pl := range pls {
		log.Debug(r.ctx, "Cleaning-up orphan tracks from playlist", "id", pl.Id, "name", pl.Name)
		del := Delete("playlist_tracks").Where(And{
			ConcatExpr("media_file_id not in (select id from media_file)"),
			Eq{"item_type": string(model.PlaylistTrackSong)},
			Eq{"playlist_id": pl.Id},
		})
		n, err := r.executeSQL(del)
		if n == 0 || err != nil {
			return fmt.Errorf("deleting orphan tracks from playlist %s: %w", pl.Name, err)
		}
		log.Debug(r.ctx, "Deleted tracks, now reordering", "id", pl.Id, "name", pl.Name, "deleted", n)

		// Renumber the playlist if any track was removed
		if err := r.renumber(pl.Id); err != nil {
			return fmt.Errorf("renumbering playlist %s: %w", pl.Name, err)
		}
	}
	return nil
}

// renumber updates the position of all tracks in the playlist to be sequential starting from 1, ordered by their
// current position. This is needed after removing orphan tracks, to ensure there are no gaps in the track numbering.
// The two-step approach (negate then reassign via CTE) avoids UNIQUE constraint violations on (playlist_id, id).
func (r *playlistRepository) renumber(id string) error {
	// Step 1: Negate all IDs to clear the positive ID space
	_, err := r.executeSQL(Expr(
		`UPDATE playlist_tracks SET id = -id WHERE playlist_id = ? AND id > 0`, id))
	if err != nil {
		return err
	}
	// Step 2: Assign new sequential positive IDs using UPDATE...FROM with a CTE.
	// The CTE is fully materialized before the UPDATE begins, avoiding self-referencing issues.
	// ORDER BY id DESC restores original order since IDs are now negative.
	_, err = r.executeSQL(Expr(
		`WITH new_ids AS (
			SELECT rowid as rid, ROW_NUMBER() OVER (ORDER BY id DESC) as new_id
			FROM playlist_tracks WHERE playlist_id = ?
		)
		UPDATE playlist_tracks SET id = new_ids.new_id
		FROM new_ids
		WHERE playlist_tracks.rowid = new_ids.rid AND playlist_tracks.playlist_id = ?`, id, id))
	if err != nil {
		return err
	}
	return r.refreshCounters(&model.Playlist{ID: id})
}

// RemoveItemFromPlaylists deletes every playlist_tracks row referencing
// itemID across all playlists (used to cascade a podcast episode's download
// deletion out of any playlist it was added to), renumbering each affected
// playlist so track positions stay contiguous.
func (r *playlistRepository) RemoveItemFromPlaylists(itemID string) error {
	sel := Select("distinct playlist_id").From("playlist_tracks").Where(Eq{"media_file_id": itemID})
	var playlistIds []string
	if err := r.queryAllSlice(sel, &playlistIds); err != nil {
		return fmt.Errorf("finding playlists referencing item %s: %w", itemID, err)
	}
	if len(playlistIds) == 0 {
		return nil
	}

	del := Delete("playlist_tracks").Where(And{Eq{"media_file_id": itemID}, Eq{"playlist_id": playlistIds}})
	if _, err := r.executeSQL(del); err != nil {
		return fmt.Errorf("removing item %s from playlists: %w", itemID, err)
	}

	for _, id := range playlistIds {
		if err := r.renumber(id); err != nil {
			return fmt.Errorf("renumbering playlist %s: %w", id, err)
		}
	}
	return nil
}

var _ model.PlaylistRepository = (*playlistRepository)(nil)
var _ rest.Repository = (*playlistRepository)(nil)
var _ rest.Persistable = (*playlistRepository)(nil)
