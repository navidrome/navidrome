package tageditor

import (
	"context"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/Masterminds/squirrel"
	"github.com/navidrome/navidrome/model"
	"go.senan.xyz/taglib"
)

type Service struct {
	ds      model.DataStore
	scanner model.Scanner
}

func New(ds model.DataStore, scanner model.Scanner) *Service {
	return &Service{ds: ds, scanner: scanner}
}

type SongPayload struct {
	ID           string `json:"id"`
	Path         string `json:"path,omitempty"`
	Title        string `json:"title"`
	Artist       string `json:"artist"`
	Album        string `json:"album"`
	AlbumArtist  string `json:"albumArtist"`
	TrackNumber  string `json:"trackNumber"`
	DiscNumber   string `json:"discNumber"`
	Date         string `json:"date"`
	ReleaseDate  string `json:"releaseDate"`
	OriginalDate string `json:"originalDate"`
	Genre        string `json:"genre"`
	Comment      string `json:"comment"`
	Lyrics       string `json:"lyrics"`
}

type AlbumPayload struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	AlbumArtist  string `json:"albumArtist"`
	Date         string `json:"date"`
	ReleaseDate  string `json:"releaseDate"`
	OriginalDate string `json:"originalDate"`
	Genre        string `json:"genre"`
	Comment      string `json:"comment"`
	Compilation  bool   `json:"compilation"`
	SongCount    int    `json:"songCount,omitempty"`
}

func (s *Service) GetSong(ctx context.Context, id string) (*SongPayload, error) {
	mf, err := s.ds.MediaFile(ctx).Get(id)
	if err != nil {
		return nil, fmt.Errorf("get song %s: %w", id, err)
	}
	return songToPayload(*mf), nil
}

func (s *Service) UpdateSong(ctx context.Context, id string, payload SongPayload) (*SongPayload, error) {
	mf, err := s.ds.MediaFile(ctx).Get(id)
	if err != nil {
		return nil, fmt.Errorf("get song %s: %w", id, err)
	}
	if mf.Missing {
		return nil, fmt.Errorf("cannot edit tags for missing file: %s", mf.Path)
	}

	tags, err := taglib.ReadTags(mf.AbsolutePath())
	if err != nil {
		return nil, fmt.Errorf("read tags from %s: %w", mf.Path, err)
	}

	applySongPayload(tags, payload)

	if err := taglib.WriteTags(mf.AbsolutePath(), tags, taglib.Clear); err != nil {
		return nil, fmt.Errorf("write tags to %s: %w", mf.Path, err)
	}

	if _, err := s.scanner.ScanFolders(ctx, false, []model.ScanTarget{scanTargetForMediaFile(*mf)}); err != nil {
		return nil, fmt.Errorf("rescan song folder: %w", err)
	}

	refreshed, err := s.reloadSongByPath(ctx, mf.Path)
	if err == nil {
		return songToPayload(refreshed), nil
	}
	return &payload, nil
}

func (s *Service) GetAlbum(ctx context.Context, id string) (*AlbumPayload, error) {
	album, err := s.ds.Album(ctx).Get(id)
	if err != nil {
		return nil, fmt.Errorf("get album %s: %w", id, err)
	}
	return albumToPayload(*album), nil
}

func (s *Service) UpdateAlbum(ctx context.Context, id string, payload AlbumPayload) (*AlbumPayload, error) {
	_, err := s.ds.Album(ctx).Get(id)
	if err != nil {
		return nil, fmt.Errorf("get album %s: %w", id, err)
	}

	tracks, err := s.ds.MediaFile(ctx).GetAll(model.QueryOptions{
		Filters: squirrel.Eq{"album_id": id},
	})
	if err != nil {
		return nil, fmt.Errorf("load album tracks: %w", err)
	}
	if len(tracks) == 0 {
		return nil, fmt.Errorf("album %s has no tracks", id)
	}

	targetMap := map[model.ScanTarget]struct{}{}
	paths := make([]string, 0, len(tracks))

	for _, mf := range tracks {
		if mf.Missing {
			continue
		}
		tags, err := taglib.ReadTags(mf.AbsolutePath())
		if err != nil {
			return nil, fmt.Errorf("read tags from %s: %w", mf.Path, err)
		}
		applyAlbumPayload(tags, payload)
		if err := taglib.WriteTags(mf.AbsolutePath(), tags, taglib.Clear); err != nil {
			return nil, fmt.Errorf("write tags to %s: %w", mf.Path, err)
		}
		targetMap[scanTargetForMediaFile(mf)] = struct{}{}
		paths = append(paths, mf.Path)
	}

	targets := make([]model.ScanTarget, 0, len(targetMap))
	for target := range targetMap {
		targets = append(targets, target)
	}
	if len(targets) > 0 {
		if _, err := s.scanner.ScanFolders(ctx, false, targets); err != nil {
			return nil, fmt.Errorf("rescan album folders: %w", err)
		}
	}

	if refreshed, err := s.reloadAlbumByPaths(ctx, paths); err == nil {
		return refreshed, nil
	}

	payload.ID = id
	payload.SongCount = len(tracks)
	return &payload, nil
}

func (s *Service) reloadSongByPath(ctx context.Context, relPath string) (model.MediaFile, error) {
	files, err := s.ds.MediaFile(ctx).FindByPaths([]string{relPath})
	if err != nil {
		return model.MediaFile{}, err
	}
	if len(files) == 0 {
		return model.MediaFile{}, fmt.Errorf("song not found after rescan: %s", relPath)
	}
	return files[0], nil
}

func (s *Service) reloadAlbumByPaths(ctx context.Context, relPaths []string) (*AlbumPayload, error) {
	files, err := s.ds.MediaFile(ctx).FindByPaths(relPaths)
	if err != nil {
		return nil, err
	}
	if len(files) == 0 {
		return nil, fmt.Errorf("album not found after rescan")
	}
	albumID := files[0].AlbumID
	return s.GetAlbum(ctx, albumID)
}

func songToPayload(mf model.MediaFile) *SongPayload {
	return &SongPayload{
		ID:           mf.ID,
		Path:         mf.Path,
		Title:        mf.Title,
		Artist:       joinValues(trackArtistValues(mf)),
		Album:        mf.Album,
		AlbumArtist:  joinValues(albumArtistValues(mf)),
		TrackNumber:  intToString(mf.TrackNumber),
		DiscNumber:   intToString(mf.DiscNumber),
		Date:         firstNonEmpty(mf.Date, firstTagValue(mf.Tags, model.TagRecordingDate)),
		ReleaseDate:  firstNonEmpty(mf.ReleaseDate, firstTagValue(mf.Tags, model.TagReleaseDate)),
		OriginalDate: firstNonEmpty(mf.OriginalDate, firstTagValue(mf.Tags, model.TagOriginalDate)),
		Genre:        joinValues(tagValues(mf.Tags, model.TagGenre, mf.Genre)),
		Comment:      mf.Comment,
		Lyrics:       mf.Lyrics,
	}
}

func albumToPayload(album model.Album) *AlbumPayload {
	return &AlbumPayload{
		ID:           album.ID,
		Name:         album.Name,
		AlbumArtist:  joinValues(albumLevelArtistValues(album)),
		Date:         firstNonEmpty(album.Date),
		ReleaseDate:  firstNonEmpty(album.ReleaseDate, firstTagValue(album.Tags, model.TagReleaseDate)),
		OriginalDate: firstNonEmpty(album.OriginalDate, firstTagValue(album.Tags, model.TagOriginalDate)),
		Genre:        joinValues(tagValues(album.Tags, model.TagGenre, album.Genre)),
		Comment:      album.Comment,
		Compilation:  album.Compilation,
		SongCount:    album.SongCount,
	}
}

func applySongPayload(tags map[string][]string, payload SongPayload) {
	setOrDelete(tags, taglib.Title, splitValues(payload.Title))
	setOrDelete(tags, taglib.Artist, splitValues(payload.Artist))
	setOrDelete(tags, taglib.Artists, splitValues(payload.Artist))
	setOrDelete(tags, taglib.Album, splitValues(payload.Album))
	setOrDelete(tags, taglib.AlbumArtist, splitValues(payload.AlbumArtist))
	setOrDelete(tags, taglib.TrackNumber, splitValues(payload.TrackNumber))
	setOrDelete(tags, taglib.DiscNumber, splitValues(payload.DiscNumber))
	setOrDelete(tags, taglib.Date, splitValues(payload.Date))
	setOrDelete(tags, taglib.ReleaseDate, splitValues(payload.ReleaseDate))
	setOrDelete(tags, taglib.OriginalDate, splitValues(payload.OriginalDate))
	setOrDelete(tags, taglib.Genre, splitValues(payload.Genre))
	setOrDelete(tags, taglib.Comment, splitValues(payload.Comment))
	setOrDelete(tags, taglib.Lyrics, splitValues(payload.Lyrics))
}

func applyAlbumPayload(tags map[string][]string, payload AlbumPayload) {
	setOrDelete(tags, taglib.Album, splitValues(payload.Name))
	setOrDelete(tags, taglib.AlbumArtist, splitValues(payload.AlbumArtist))
	setOrDelete(tags, taglib.Date, splitValues(payload.Date))
	setOrDelete(tags, taglib.ReleaseDate, splitValues(payload.ReleaseDate))
	setOrDelete(tags, taglib.OriginalDate, splitValues(payload.OriginalDate))
	setOrDelete(tags, taglib.Genre, splitValues(payload.Genre))
	setOrDelete(tags, taglib.Comment, splitValues(payload.Comment))

	if payload.Compilation {
		setOrDelete(tags, taglib.Compilation, []string{"1"})
	} else {
		delete(tags, taglib.Compilation)
	}
}

func scanTargetForMediaFile(mf model.MediaFile) model.ScanTarget {
	dir := filepath.Dir(mf.Path)
	if dir == "." {
		dir = ""
	}
	return model.ScanTarget{
		LibraryID:  mf.LibraryID,
		FolderPath: dir,
	}
}

func tagValues(tags model.Tags, key model.TagName, fallback string) []string {
	values := tags[key]
	if len(values) > 0 {
		return values
	}
	if fallback == "" {
		return nil
	}
	return []string{fallback}
}

func trackArtistValues(mf model.MediaFile) []string {
	if values := mf.Tags[model.TagTrackArtists]; len(values) > 0 {
		return values
	}
	if values := mf.Tags[model.TagTrackArtist]; len(values) > 0 {
		return values
	}
	if mf.Artist == "" {
		return nil
	}
	return []string{mf.Artist}
}

func albumArtistValues(mf model.MediaFile) []string {
	if values := mf.Tags[model.TagAlbumArtists]; len(values) > 0 {
		return values
	}
	if values := mf.Tags[model.TagAlbumArtist]; len(values) > 0 {
		return values
	}
	if mf.AlbumArtist == "" {
		return nil
	}
	return []string{mf.AlbumArtist}
}

func albumLevelArtistValues(album model.Album) []string {
	if values := album.Tags[model.TagAlbumArtists]; len(values) > 0 {
		return values
	}
	if values := album.Tags[model.TagAlbumArtist]; len(values) > 0 {
		return values
	}
	if album.AlbumArtist == "" {
		return nil
	}
	return []string{album.AlbumArtist}
}

func firstTagValue(tags model.Tags, keys ...model.TagName) string {
	for _, key := range keys {
		if values := tags[key]; len(values) > 0 && strings.TrimSpace(values[0]) != "" {
			return strings.TrimSpace(values[0])
		}
	}
	return ""
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func joinValues(values []string) string {
	if len(values) == 0 {
		return ""
	}
	return strings.Join(values, "; ")
}

func setOrDelete(tags map[string][]string, key string, values []string) {
	if len(values) == 0 {
		delete(tags, key)
		return
	}
	tags[key] = values
}

func splitValues(value string) []string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}

	raw := strings.FieldsFunc(value, func(r rune) bool {
		return r == ';' || r == '\n'
	})
	values := make([]string, 0, len(raw))
	seen := map[string]struct{}{}
	for _, item := range raw {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		values = append(values, item)
	}
	return values
}

func intToString(v int) string {
	if v == 0 {
		return ""
	}
	return strconv.Itoa(v)
}
