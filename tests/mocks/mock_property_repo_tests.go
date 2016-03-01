package mocks

import (
	"github.com/deluan/gosonic/repositories"
	"errors"
)

func CreateMockPropertyRepo() *MockProperty {
	return &MockProperty{data: make(map[string]string)}
}

type MockProperty struct {
	repositories.PropertyImpl
	data map[string]string
	err  bool
}

func (m *MockProperty) SetError(err bool) {
	m.err = err
}

func (m *MockProperty) Put(id string, value string) error {
	if m.err {
		return errors.New("Error!")
	}
	m.data[id] = value
	return nil
}

func (m *MockProperty) Get(id string) (string, error) {
	if m.err {
		return "", errors.New("Error!")
	} else {
		return m.data[id], nil
	}
}

func (m *MockProperty) DefaultGet(id string, defaultValue string) (string, error) {
	v, err := m.Get(id)

	if v == "" {
		v = defaultValue
	}

	return v, err
}
