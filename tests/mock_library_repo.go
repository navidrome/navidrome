package tests

import (
	"errors"
	"slices"
	"strconv"

	"github.com/Masterminds/squirrel"
	"github.com/deluan/rest"
	"github.com/navidrome/navidrome/model"
)

type MockLibraryRepo struct {
	model.LibraryRepository
	Data map[int]model.Library
	Err  error
}

func (m *MockLibraryRepo) SetData(data model.Libraries) {
	m.Data = make(map[int]model.Library)
	for _, d := range data {
		m.Data[d.ID] = d
	}
}

func (m *MockLibraryRepo) GetAll(...model.QueryOptions) (model.Libraries, error) {
	if m.Err != nil {
		return nil, m.Err
	}
	var libraries model.Libraries
	for _, lib := range m.Data {
		libraries = append(libraries, lib)
	}
	// Sort by ID for predictable order
	slices.SortFunc(libraries, func(a, b model.Library) int {
		return a.ID - b.ID
	})
	return libraries, nil
}

func (m *MockLibraryRepo) CountAll(qo ...model.QueryOptions) (int64, error) {
	if m.Err != nil {
		return 0, m.Err
	}

	// If no query options, return total count
	if len(qo) == 0 || qo[0].Filters == nil {
		return int64(len(m.Data)), nil
	}

	// Handle squirrel.Eq filter for ID validation
	if eq, ok := qo[0].Filters.(squirrel.Eq); ok {
		if idFilter, exists := eq["id"]; exists {
			if ids, isSlice := idFilter.([]int); isSlice {
				count := 0
				for _, id := range ids {
					if _, exists := m.Data[id]; exists {
						count++
					}
				}
				return int64(count), nil
			}
		}
	}

	// Default to total count for other filters
	return int64(len(m.Data)), nil
}

func (m *MockLibraryRepo) Get(id int) (*model.Library, error) {
	if m.Err != nil {
		return nil, m.Err
	}
	if lib, ok := m.Data[id]; ok {
		return &lib, nil
	}
	return nil, model.ErrNotFound
}

func (m *MockLibraryRepo) GetPath(id int) (string, error) {
	if m.Err != nil {
		return "", m.Err
	}
	if lib, ok := m.Data[id]; ok {
		return lib.Path, nil
	}
	return "", model.ErrNotFound
}

func (m *MockLibraryRepo) Put(library *model.Library) error {
	if m.Err != nil {
		return m.Err
	}
	if m.Data == nil {
		m.Data = make(map[int]model.Library)
	}
	m.Data[library.ID] = *library
	return nil
}

func (m *MockLibraryRepo) Delete(id int) error {
	if m.Err != nil {
		return m.Err
	}
	if _, ok := m.Data[id]; !ok {
		return model.ErrNotFound
	}
	delete(m.Data, id)
	return nil
}

func (m *MockLibraryRepo) StoreMusicFolder() error {
	if m.Err != nil {
		return m.Err
	}
	return nil
}

func (m *MockLibraryRepo) AddArtist(id int, artistID string) error {
	if m.Err != nil {
		return m.Err
	}
	return nil
}

func (m *MockLibraryRepo) ScanBegin(id int, fullScan bool) error {
	if m.Err != nil {
		return m.Err
	}
	return nil
}

func (m *MockLibraryRepo) ScanEnd(id int) error {
	if m.Err != nil {
		return m.Err
	}
	return nil
}

func (m *MockLibraryRepo) ScanInProgress() (bool, error) {
	if m.Err != nil {
		return false, m.Err
	}
	return false, nil
}

func (m *MockLibraryRepo) RefreshStats(id int) error {
	return nil
}

// User-library association methods - mock implementations

func (m *MockLibraryRepo) GetUsersWithLibraryAccess(libraryID int) (model.Users, error) {
	if m.Err != nil {
		return nil, m.Err
	}
	// Mock: return empty users for now
	return model.Users{}, nil
}

func (m *MockLibraryRepo) Count(options ...rest.QueryOptions) (int64, error) {
	return m.CountAll()
}

func (m *MockLibraryRepo) Read(id string) (interface{}, error) {
	idInt, _ := strconv.Atoi(id)
	mf, err := m.Get(idInt)
	if errors.Is(err, model.ErrNotFound) {
		return nil, rest.ErrNotFound
	}
	return mf, err
}

func (m *MockLibraryRepo) ReadAll(options ...rest.QueryOptions) (interface{}, error) {
	return m.GetAll()
}

func (m *MockLibraryRepo) EntityName() string {
	return "library"
}

func (m *MockLibraryRepo) NewInstance() interface{} {
	return &model.Library{}
}

var _ model.LibraryRepository = (*MockLibraryRepo)(nil)
var _ model.ResourceRepository = (*MockLibraryRepo)(nil)
