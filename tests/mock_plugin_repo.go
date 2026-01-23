package tests

import (
	"errors"
	"time"

	"github.com/deluan/rest"
	"github.com/navidrome/navidrome/model"
)

func CreateMockPluginRepo() *MockPluginRepo {
	return &MockPluginRepo{
		Data:      make(map[string]*model.Plugin),
		IsAdmin:   true, // Default to admin access
		Permitted: true,
	}
}

type MockPluginRepo struct {
	Data      map[string]*model.Plugin
	All       model.Plugins
	Err       bool
	Options   model.QueryOptions
	IsAdmin   bool
	Permitted bool
}

func (m *MockPluginRepo) SetError(err bool) {
	m.Err = err
}

func (m *MockPluginRepo) SetData(plugins model.Plugins) {
	m.Data = make(map[string]*model.Plugin, len(plugins))
	m.All = plugins
	for i, p := range m.All {
		m.Data[p.ID] = &m.All[i]
	}
}

func (m *MockPluginRepo) SetPermitted(permitted bool) {
	m.Permitted = permitted
}

func (m *MockPluginRepo) Get(id string) (*model.Plugin, error) {
	if !m.Permitted {
		return nil, rest.ErrPermissionDenied
	}
	if m.Err {
		return nil, errors.New("unexpected error")
	}
	if d, ok := m.Data[id]; ok {
		return d, nil
	}
	return nil, model.ErrNotFound
}

func (m *MockPluginRepo) Read(id string) (interface{}, error) {
	p, err := m.Get(id)
	if errors.Is(err, model.ErrNotFound) {
		return nil, rest.ErrNotFound
	}
	return p, err
}

func (m *MockPluginRepo) Put(p *model.Plugin) error {
	if !m.Permitted {
		return rest.ErrPermissionDenied
	}
	if m.Err {
		return errors.New("unexpected error")
	}
	if p.ID == "" {
		return errors.New("plugin ID cannot be empty")
	}
	now := time.Now()
	if existing, ok := m.Data[p.ID]; ok {
		p.CreatedAt = existing.CreatedAt
	} else {
		p.CreatedAt = now
	}
	p.UpdatedAt = now
	m.Data[p.ID] = p
	// Update All slice
	found := false
	for i, existing := range m.All {
		if existing.ID == p.ID {
			m.All[i] = *p
			found = true
			break
		}
	}
	if !found {
		m.All = append(m.All, *p)
	}
	return nil
}

func (m *MockPluginRepo) Delete(id string) error {
	if !m.Permitted {
		return rest.ErrPermissionDenied
	}
	if m.Err {
		return errors.New("unexpected error")
	}
	delete(m.Data, id)
	// Update All slice
	for i, p := range m.All {
		if p.ID == id {
			m.All = append(m.All[:i], m.All[i+1:]...)
			break
		}
	}
	return nil
}

func (m *MockPluginRepo) GetAll(qo ...model.QueryOptions) (model.Plugins, error) {
	if len(qo) > 0 {
		m.Options = qo[0]
	}
	if !m.Permitted {
		return nil, rest.ErrPermissionDenied
	}
	if m.Err {
		return nil, errors.New("unexpected error")
	}
	return m.All, nil
}

func (m *MockPluginRepo) CountAll(qo ...model.QueryOptions) (int64, error) {
	if len(qo) > 0 {
		m.Options = qo[0]
	}
	if !m.Permitted {
		return 0, rest.ErrPermissionDenied
	}
	if m.Err {
		return 0, errors.New("unexpected error")
	}
	return int64(len(m.All)), nil
}

// rest.Repository interface methods
func (m *MockPluginRepo) Count(options ...rest.QueryOptions) (int64, error) {
	if !m.Permitted {
		return 0, rest.ErrPermissionDenied
	}
	return int64(len(m.All)), nil
}

func (m *MockPluginRepo) EntityName() string {
	return "plugin"
}

func (m *MockPluginRepo) NewInstance() interface{} {
	return &model.Plugin{}
}

func (m *MockPluginRepo) ReadAll(options ...rest.QueryOptions) (interface{}, error) {
	return m.GetAll()
}

func (m *MockPluginRepo) Save(entity interface{}) (string, error) {
	p := entity.(*model.Plugin)
	err := m.Put(p)
	return p.ID, err
}

func (m *MockPluginRepo) Update(id string, entity interface{}, cols ...string) error {
	p := entity.(*model.Plugin)
	p.ID = id
	return m.Put(p)
}

var _ model.PluginRepository = (*MockPluginRepo)(nil)
