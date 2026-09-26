package model

import (
	"context"
	"database/sql/driver"
	"fmt"
	"strings"
	"time"
)

type Grant struct {
	ID            string     `structs:"id" json:"id"`
	UserID        string     `structs:"user_id" json:"userId"`
	Name          string     `structs:"name" json:"name"`
	Client        string     `structs:"client" json:"client"`
	ClientVersion string     `structs:"client_version" json:"clientVersion"`
	Scopes        Scopes     `structs:"scopes" json:"scopes"`
	Provider      string     `structs:"provider" json:"provider"`
	SecretHash    string     `structs:"secret_hash" json:"-"`
	UserEpoch     int        `structs:"user_epoch" json:"-"`
	CreatedAt     time.Time  `structs:"created_at" json:"createdAt"`
	LastUsedAt    *time.Time `structs:"last_used_at" json:"lastUsedAt"`
	LastUsedIP    string     `structs:"last_used_ip" json:"lastUsedIp"`
}

type Grants []Grant

// Scopes is stored as a single space-separated column.
type Scopes []string

func (s Scopes) Value() (driver.Value, error) {
	return strings.Join(s, " "), nil
}

func (s *Scopes) Scan(src any) error {
	switch v := src.(type) {
	case string:
		*s = strings.Fields(v)
	case []byte:
		*s = strings.Fields(string(v))
	case nil:
		*s = nil
	default:
		return fmt.Errorf("cannot scan %T into Scopes", src)
	}
	return nil
}

type GrantRepository interface {
	Put(ctx context.Context, g *Grant) error
	Get(ctx context.Context, id string) (*Grant, error)
	FindBySecretHash(ctx context.Context, hash string) (*Grant, error)
	GetAllForUser(ctx context.Context, userID string, epoch int, idleSince time.Time, offset, limit int) (Grants, error)
	CountForUser(ctx context.Context, userID string, epoch int, idleSince time.Time) (int64, error)
	Delete(ctx context.Context, id string) error
	DeleteForUser(ctx context.Context, userID, id string) error
	DeleteOtherEpochs(ctx context.Context, userID string, epoch int) error
	SetEpoch(ctx context.Context, userID string, fromEpoch, toEpoch int, onlyID string) error
	DeleteIfEpoch(ctx context.Context, id string, epoch int) error
	Touch(ctx context.Context, id, ip string, at, notSince time.Time) error
	DeleteIdle(ctx context.Context, idleSince time.Time) (int64, error)
}
