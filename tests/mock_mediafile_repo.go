package tests

import (
	"cmp"
	"context"
	"errors"
	"maps"
	"slices"
	"time"

	"github.com/deluan/rest"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/criteria"
	"github.com/navidrome/navidrome/model/id"
	"github.com/navidrome/navidrome/utils/slice"
)

func CreateMockMediaFileRepo() *MockMediaFileRepo {
	return &MockMediaFileRepo{
		Data: make(map[string]*model.MediaFile),
	}
}

type MockMediaFileRepo struct {
	model.MediaFileRepository
	Data map[string]*model.MediaFile
	Err  bool
	// Add fields and methods for controlling CountAll and DeleteAllMissing in tests.
	// A nil CountAllValue is unset, and CountAll falls back to counting rows in Data.
	CountAllValue         *int64
	CountAllOptions       model.QueryOptions
	DeleteAllMissingValue int64
	// ReassignReferencesCalls records prevID -> newID
	ReassignReferencesCalls map[string]string
	Options                 model.QueryOptions
	// Add fields for cross-library move detection tests
	FindRecentFilesByMBZTrackIDFunc func(missing model.MediaFile, since time.Time) (model.MediaFiles, error)
	FindRecentFilesByPropertiesFunc func(missing model.MediaFile, since time.Time) (model.MediaFiles, error)
	MatchesCriteriaValue            bool
	MatchesCriteriaErr              error
	BookmarksAdded                  []string
}

func (m *MockMediaFileRepo) SetError(err bool) {
	m.Err = err
}

func (m *MockMediaFileRepo) SetCountAll(count int64) {
	m.CountAllValue = &count
}

func (m *MockMediaFileRepo) SetData(mfs model.MediaFiles) {
	m.Data = make(map[string]*model.MediaFile)
	for i, mf := range mfs {
		m.Data[mf.ID] = &mfs[i]
	}
}

func (m *MockMediaFileRepo) Exists(_ context.Context, id string) (bool, error) {
	if m.Err {
		return false, errors.New("error")
	}
	_, found := m.Data[id]
	return found, nil
}

func (m *MockMediaFileRepo) Get(_ context.Context, id string) (*model.MediaFile, error) {
	if m.Err {
		return nil, errors.New("error")
	}
	if d, ok := m.Data[id]; ok {
		// Intentionally clone the file and remove participants. This should
		// catch any caller that actually means to call GetWithParticipants
		res := *d
		res.Participants = model.Participants{}
		return &res, nil
	}
	return nil, model.ErrNotFound
}

func (m *MockMediaFileRepo) AddBookmark(_ context.Context, id, _ string, _ int64) error {
	if m.Err {
		return errors.New("error")
	}
	m.BookmarksAdded = append(m.BookmarksAdded, id)
	return nil
}

func (m *MockMediaFileRepo) GetWithParticipants(_ context.Context, id string) (*model.MediaFile, error) {
	if m.Err {
		return nil, errors.New("error")
	}
	if d, ok := m.Data[id]; ok {
		return d, nil
	}
	return nil, model.ErrNotFound
}

func (m *MockMediaFileRepo) GetAllByTags(ctx context.Context, _ model.TagName, _ []string, options ...model.QueryOptions) (model.MediaFiles, error) {
	return m.GetAll(ctx, options...)
}

func (m *MockMediaFileRepo) GetAll(_ context.Context, qo ...model.QueryOptions) (model.MediaFiles, error) {
	if len(qo) > 0 {
		m.Options = qo[0]
	}
	if m.Err {
		return nil, errors.New("error")
	}
	values := slices.Collect(maps.Values(m.Data))
	result := slice.Map(values, func(p *model.MediaFile) model.MediaFile {
		return *p
	})
	// Sort by ID to ensure deterministic ordering for tests
	slices.SortFunc(result, func(a, b model.MediaFile) int {
		return cmp.Compare(a.ID, b.ID)
	})
	return result, nil
}

func (m *MockMediaFileRepo) GetRandom(ctx context.Context, qo ...model.QueryOptions) (model.MediaFiles, error) {
	res, err := m.GetAll(ctx, qo...)
	if err != nil {
		return nil, err
	}
	if len(qo) > 0 && qo[0].Max > 0 && len(res) > qo[0].Max {
		res = res[:qo[0].Max]
	}
	return res, nil
}

func (m *MockMediaFileRepo) GetCursor(ctx context.Context, qo ...model.QueryOptions) (model.MediaFileCursor, error) {
	res, err := m.GetAll(ctx, qo...)
	if err != nil {
		return nil, err
	}
	return func(yield func(model.MediaFile, error) bool) {
		for _, mf := range res {
			if !yield(mf, nil) {
				return
			}
		}
	}, nil
}

func (m *MockMediaFileRepo) GetCursorWithArtwork(ctx context.Context, qo ...model.QueryOptions) (model.MediaFileCursor, error) {
	return m.GetCursor(ctx, qo...)
}

func (m *MockMediaFileRepo) Put(_ context.Context, mf *model.MediaFile) error {
	if m.Err {
		return errors.New("error")
	}
	if mf.ID == "" {
		mf.ID = id.NewRandom()
	}
	m.Data[mf.ID] = mf
	return nil
}

func (m *MockMediaFileRepo) UpdateProbeData(_ context.Context, id string, data string) error {
	if m.Err {
		return errors.New("error")
	}
	if d, ok := m.Data[id]; ok {
		d.ProbeData = data
		return nil
	}
	return model.ErrNotFound
}

func (m *MockMediaFileRepo) Delete(_ context.Context, id string) error {
	if m.Err {
		return errors.New("error")
	}
	if _, ok := m.Data[id]; !ok {
		return model.ErrNotFound
	}
	delete(m.Data, id)
	return nil
}

func (m *MockMediaFileRepo) ReassignReferences(_ context.Context, prevID, newID string) error {
	if m.Err {
		return errors.New("error")
	}
	if m.ReassignReferencesCalls == nil {
		m.ReassignReferencesCalls = make(map[string]string)
	}
	m.ReassignReferencesCalls[prevID] = newID
	return nil
}

func (m *MockMediaFileRepo) IncPlayCount(_ context.Context, id string, timestamp time.Time) error {
	if m.Err {
		return errors.New("error")
	}
	if d, ok := m.Data[id]; ok {
		d.PlayCount++
		d.PlayDate = &timestamp
		return nil
	}
	return model.ErrNotFound
}

func (m *MockMediaFileRepo) SetStar(_ context.Context, starred bool, itemIDs ...string) error {
	if m.Err {
		return errors.New("error")
	}
	for _, id := range itemIDs {
		if d, ok := m.Data[id]; ok {
			d.Starred = starred
		}
	}
	return nil
}

func (m *MockMediaFileRepo) SetRating(_ context.Context, rating int, itemID string) error {
	if m.Err {
		return errors.New("error")
	}
	if d, ok := m.Data[itemID]; ok {
		d.Rating = rating
	}
	return nil
}

func (m *MockMediaFileRepo) FindByAlbum(artistId string) (model.MediaFiles, error) {
	if m.Err {
		return nil, errors.New("error")
	}
	var res = make(model.MediaFiles, len(m.Data))
	i := 0
	for _, a := range m.Data {
		if a.AlbumID == artistId {
			res[i] = *a
			i++
		}
	}

	return res, nil
}

func (m *MockMediaFileRepo) GetMissingAndMatching(_ context.Context, libId int) (model.MediaFileCursor, error) {
	if m.Err {
		return nil, errors.New("error")
	}
	var res model.MediaFiles
	for _, a := range m.Data {
		if a.LibraryID == libId && a.Missing {
			res = append(res, *a)
		}
	}

	for _, a := range m.Data {
		if a.LibraryID == libId && !(*a).Missing && slices.IndexFunc(res, func(mediaFile model.MediaFile) bool {
			return mediaFile.PID == a.PID
		}) != -1 {
			res = append(res, *a)
		}
	}
	slices.SortFunc(res, func(i, j model.MediaFile) int {
		return cmp.Or(
			cmp.Compare(i.PID, j.PID),
			cmp.Compare(i.ID, j.ID),
		)
	})

	return func(yield func(model.MediaFile, error) bool) {
		for _, a := range res {
			if !yield(a, nil) {
				break
			}
		}
	}, nil
}

func (m *MockMediaFileRepo) CountAll(_ context.Context, opts ...model.QueryOptions) (int64, error) {
	if m.Err {
		return 0, errors.New("error")
	}
	if len(opts) > 0 {
		m.CountAllOptions = opts[0]
	}
	if m.CountAllValue != nil {
		return *m.CountAllValue, nil
	}
	return int64(len(m.Data)), nil
}

func (m *MockMediaFileRepo) DeleteAllMissing(_ context.Context) (int64, error) {
	if m.Err {
		return 0, errors.New("error")
	}
	if m.DeleteAllMissingValue != 0 {
		return m.DeleteAllMissingValue, nil
	}
	// Remove all missing files from Data
	var count int64
	for id, mf := range m.Data {
		if mf.Missing {
			delete(m.Data, id)
			count++
		}
	}
	return count, nil
}

// REST repository methods
func (m *MockMediaFileRepo) Count(ctx context.Context, _ ...rest.QueryOptions) (int64, error) {
	return m.CountAll(ctx)
}

func (m *MockMediaFileRepo) Read(ctx context.Context, id string) (*model.MediaFile, error) {
	return m.Get(ctx, id)
}

func (m *MockMediaFileRepo) ReadAll(ctx context.Context, _ ...rest.QueryOptions) ([]model.MediaFile, error) {
	return m.GetAll(ctx)
}

func (m *MockMediaFileRepo) Search(ctx context.Context, q string, options ...model.QueryOptions) (model.MediaFiles, error) {
	if len(options) > 0 {
		m.Options = options[0]
	}
	if m.Err {
		return nil, errors.New("unexpected error")
	}
	// Simple mock implementation - just return all media files for testing
	return m.GetAll(ctx)
}

// Cross-library move detection mock methods
func (m *MockMediaFileRepo) FindRecentFilesByMBZTrackID(_ context.Context, missing model.MediaFile, since time.Time) (model.MediaFiles, error) {
	if m.Err {
		return nil, errors.New("error")
	}
	if m.FindRecentFilesByMBZTrackIDFunc != nil {
		return m.FindRecentFilesByMBZTrackIDFunc(missing, since)
	}
	// Default implementation: find files with same MBZ Track ID in other libraries
	var result model.MediaFiles
	for _, mf := range m.Data {
		if mf.LibraryID != missing.LibraryID &&
			mf.MbzReleaseTrackID == missing.MbzReleaseTrackID &&
			mf.MbzReleaseTrackID != "" &&
			mf.Suffix == missing.Suffix &&
			mf.CreatedAt.After(since) &&
			!mf.Missing {
			result = append(result, *mf)
		}
	}
	return result, nil
}

func (m *MockMediaFileRepo) FindRecentFilesByProperties(_ context.Context, missing model.MediaFile, since time.Time) (model.MediaFiles, error) {
	if m.Err {
		return nil, errors.New("error")
	}
	if m.FindRecentFilesByPropertiesFunc != nil {
		return m.FindRecentFilesByPropertiesFunc(missing, since)
	}
	// Default implementation: find files with same properties in other libraries
	var result model.MediaFiles
	for _, mf := range m.Data {
		if mf.LibraryID != missing.LibraryID &&
			mf.Title == missing.Title &&
			mf.Size == missing.Size &&
			mf.Suffix == missing.Suffix &&
			mf.DiscNumber == missing.DiscNumber &&
			mf.TrackNumber == missing.TrackNumber &&
			mf.Album == missing.Album &&
			mf.MbzReleaseTrackID == "" && // Exclude files with MBZ Track ID
			mf.CreatedAt.After(since) &&
			!mf.Missing {
			result = append(result, *mf)
		}
	}
	return result, nil
}

func (m *MockMediaFileRepo) MatchesCriteria(context.Context, string, criteria.Criteria) (bool, error) {
	if m.MatchesCriteriaErr != nil {
		return false, m.MatchesCriteriaErr
	}
	return m.MatchesCriteriaValue, nil
}

var _ model.MediaFileRepository = (*MockMediaFileRepo)(nil)
var _ rest.Repository[model.MediaFile] = (*MockMediaFileRepo)(nil)
