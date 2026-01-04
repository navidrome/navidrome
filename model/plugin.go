package model

import "time"

type Plugin struct {
	ID           string    `structs:"id"            json:"id"`
	Path         string    `structs:"path"          json:"path"`
	Manifest     string    `structs:"manifest"      json:"manifest"`
	Config       string    `structs:"config"        json:"config,omitempty"`
	Users        string    `structs:"users"         json:"users,omitempty"`
	AllUsers     bool      `structs:"all_users"     json:"allUsers,omitempty"`
	Libraries    string    `structs:"libraries"     json:"libraries,omitempty"`
	AllLibraries bool      `structs:"all_libraries" json:"allLibraries,omitempty"`
	Enabled      bool      `structs:"enabled"       json:"enabled"`
	LastError    string    `structs:"last_error"    json:"lastError,omitempty"`
	SHA256       string    `structs:"sha256"        json:"sha256"`
	CreatedAt    time.Time `structs:"created_at"    json:"createdAt"`
	UpdatedAt    time.Time `structs:"updated_at"    json:"updatedAt"`
}

type Plugins []Plugin

type PluginRepository interface {
	ResourceRepository
	CountAll(options ...QueryOptions) (int64, error)
	Delete(id string) error
	Get(id string) (*Plugin, error)
	GetAll(options ...QueryOptions) (Plugins, error)
	Put(p *Plugin) error
}
