package model

import (
	"time"
)

type Share struct {
	ID            string     `structs:"id" json:"id,omitempty"           orm:"column(id)"`
	UserID        string     `structs:"user_id" json:"userId,omitempty"  orm:"column(user_id)"`
	Username      string     `structs:"-" json:"username,omitempty"      orm:"-"`
	Description   string     `structs:"description" json:"description,omitempty"`
	ExpiresAt     time.Time  `structs:"expires_at" json:"expiresAt,omitempty"`
	LastVisitedAt time.Time  `structs:"last_visited_at" json:"lastVisitedAt,omitempty"`
	ResourceIDs   string     `structs:"resource_ids" json:"resourceIds,omitempty"   orm:"column(resource_ids)"`
	ResourceType  string     `structs:"resource_type" json:"resourceType,omitempty"`
	Contents      string     `structs:"contents" json:"contents,omitempty"`
	Format        string     `structs:"format" json:"format,omitempty"`
	MaxBitRate    int        `structs:"max_bit_rate" json:"maxBitRate,omitempty"`
	VisitCount    int        `structs:"visit_count" json:"visitCount,omitempty"`
	CreatedAt     time.Time  `structs:"created_at" json:"createdAt,omitempty"`
	UpdatedAt     time.Time  `structs:"updated_at" json:"updatedAt,omitempty"`
	Tracks        MediaFiles `structs:"-" json:"tracks,omitempty"      orm:"-"`
	Albums        Albums     `structs:"-" json:"albums,omitempty"      orm:"-"`
}

type Shares []Share

type ShareRepository interface {
	Exists(id string) (bool, error)
	Get(id string) (*Share, error)
	GetAll(options ...QueryOptions) (Shares, error)
}
