package model

import (
	"time"
)

type Share struct {
	ID            string    `json:"id"            orm:"column(id)"`
	JWT           string    `json:"jwt"           orm:"column(jwt)"`
	Description   string    `json:"description"`
	ExpiresAt     time.Time `json:"expiresAt"`
	CreatedAt     time.Time `json:"createdAt"`
	LastVisitedAt time.Time `json:"lastVisitedAt"`
	ResourceID    string    `json:"resourceID"    orm:"column(resource_id)"`
	ResourceType  string    `json:"resourceType"`
	VisitCount    string    `json:"visitCount"`
}

type Shares []Share

type ShareRepository interface {
	Put(s *Share) error
	GetAll(options ...QueryOptions) (Shares, error)
}
