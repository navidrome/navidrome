package mocks

import (
	"github.com/deluan/gosonic/models"
	"fmt"
	"encoding/json"
	"github.com/deluan/gosonic/repositories"
	"errors"
)

func CreateMockArtistIndexRepo() *MockArtistIndex {
	return &MockArtistIndex{}
}

type MockArtistIndex struct {
	repositories.ArtistIndex
	data []models.ArtistIndex
	err  bool
}

func (m *MockArtistIndex) SetError(err bool) {
	m.err = err
}

func (m *MockArtistIndex) SetData(j string, length int) {
	m.data = make([]models.ArtistIndex, length)
	err := json.Unmarshal([]byte(j), &m.data)
	if err != nil {
		fmt.Println("ERROR: ", err)
	}
}

func (m *MockArtistIndex) GetAll() ([]models.ArtistIndex, error) {
	if m.err {
		return nil, errors.New("Error!")
	}
	return m.data, nil
}
