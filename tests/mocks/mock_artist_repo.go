package mocks

import (
	"encoding/json"
	"fmt"
	"github.com/deluan/gosonic/domain"
"errors"
)

func CreateMockArtistRepo() *MockArtist {
	return &MockArtist{}
}

type MockArtist struct {
	domain.ArtistRepository
	data map[string]domain.Artist
	err  bool
}

func (m *MockArtist) SetError(err bool) {
	m.err = err
}

func (m *MockArtist) SetData(j string) {
	m.data = make(map[string]domain.Artist)
	err := json.Unmarshal([]byte(j), &m.data)
	if err != nil {
		fmt.Println("ERROR: ", err)
	}
}

func (m *MockArtist) Exists(id string) (bool, error) {
	if m.err {
		return false, errors.New("Error!")
	}
	_, found := m.data[id];
	return found, nil
}