package persistence

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/deluan/gosonic/domain"
)

func CreateMockArtistRepo() *MockArtist {
	return &MockArtist{}
}

type MockArtist struct {
	domain.ArtistRepository
	data map[string]*domain.Artist
	err  bool
}

func (m *MockArtist) SetError(err bool) {
	m.err = err
}

func (m *MockArtist) SetData(j string, size int) {
	m.data = make(map[string]*domain.Artist)
	var l = make([]domain.Artist, size)
	err := json.Unmarshal([]byte(j), &l)
	if err != nil {
		fmt.Println("ERROR: ", err)
	}
	for _, a := range l {
		m.data[a.Id] = &a
	}
}

func (m *MockArtist) Exists(id string) (bool, error) {
	_, found := m.data[id]
	return found, nil
}

func (m *MockArtist) Get(id string) (*domain.Artist, error) {
	if m.err {
		return nil, errors.New("Error!")
	}
	if d, ok := m.data[id]; ok {
		return d, nil
	}
	return nil, domain.ErrNotFound
}
