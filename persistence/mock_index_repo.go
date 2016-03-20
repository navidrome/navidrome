package persistence

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/deluan/gosonic/domain"
)

func CreateMockArtistIndexRepo() *MockArtistIndex {
	return &MockArtistIndex{}
}

type MockArtistIndex struct {
	domain.ArtistIndexRepository
	data domain.ArtistIndexes
	err  bool
}

func (m *MockArtistIndex) SetError(err bool) {
	m.err = err
}

func (m *MockArtistIndex) SetData(j string, length int) {
	m.data = make(domain.ArtistIndexes, length)
	err := json.Unmarshal([]byte(j), &m.data)
	if err != nil {
		fmt.Println("ERROR: ", err)
	}
}

func (m *MockArtistIndex) GetAll() (domain.ArtistIndexes, error) {
	if m.err {
		return nil, errors.New("Error!")
	}
	return m.data, nil
}
