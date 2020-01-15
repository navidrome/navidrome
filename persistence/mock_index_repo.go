package persistence

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/cloudsonic/sonic-server/model"
)

func CreateMockArtistIndexRepo() *MockArtistIndex {
	return &MockArtistIndex{}
}

type MockArtistIndex struct {
	model.ArtistIndexRepository
	data model.ArtistIndexes
	err  bool
}

func (m *MockArtistIndex) SetError(err bool) {
	m.err = err
}

func (m *MockArtistIndex) SetData(j string, length int) {
	m.data = make(model.ArtistIndexes, length)
	err := json.Unmarshal([]byte(j), &m.data)
	if err != nil {
		fmt.Println("ERROR: ", err)
	}
}

func (m *MockArtistIndex) GetAll() (model.ArtistIndexes, error) {
	if m.err {
		return nil, errors.New("Error!")
	}
	return m.data, nil
}

var _ model.ArtistIndexRepository = (*MockArtistIndex)(nil)
