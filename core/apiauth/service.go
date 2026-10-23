package apiauth

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/core/auth"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/utils/gg"
)

const (
	IdleExpiry    = consts.APIv1GrantIdleExpiry
	touchInterval = 5 * time.Minute
)

var (
	ErrPasswordManagedExternally = errors.New("password is managed externally")
	ErrCurrentPasswordMismatch   = errors.New("current password does not match")
)

type ClientMeta struct {
	Name          string
	Client        string
	ClientVersion string
}

type Issued struct {
	Secret string
	Grant  model.Grant
	User   model.User
}

type Principal struct {
	User    model.User
	GrantID string
	Scopes  []string
}

type Service struct {
	ds       model.DataStore
	checkers func(ds model.DataStore) []CredentialChecker // per datastore, so password change can check inside its transaction
	now      func() time.Time
}

func New(ds model.DataStore) *Service {
	return &Service{
		ds: ds,
		checkers: func(ds model.DataStore) []CredentialChecker {
			return []CredentialChecker{dbChecker{ds: ds}}
		},
		now: time.Now,
	}
}

func PasswordChangeable(u model.User) bool {
	return u.IsAdmin || conf.Server.EnableUserEditing
}

func (s *Service) Login(ctx context.Context, username, password string, meta ClientMeta, scopes []string) (*Issued, error) {
	res, err := checkCredentials(ctx, s.checkers(s.ds), username, password)
	if err != nil {
		return nil, err
	}
	issued, err := s.issue(ctx, s.ds, *res.User, res.Provider, meta, scopes)
	if err != nil {
		return nil, err
	}
	if err := s.ds.User().UpdateLastLoginAt(ctx, res.User.ID); err != nil {
		log.Warn(ctx, "API v1: could not update last login", "user", res.User.UserName, err)
	}
	return issued, nil
}

func (s *Service) Setup(ctx context.Context, username, password string, meta ClientMeta, scopes []string) (*Issued, error) {
	var issued *Issued
	_, err := auth.CreateFirstAdmin(ctx, s.ds, username, password, func(tx model.DataStore, u *model.User) error {
		var err error
		issued, err = s.issue(ctx, tx, *u, "setup", meta, scopes)
		return err
	})
	if err != nil {
		return nil, err
	}
	return issued, nil
}

// issue stores a grant bound to the epoch read with the user, so a racing password change leaves it dead.
func (s *Service) issue(ctx context.Context, ds model.DataStore, u model.User, provider string, meta ClientMeta, scopes []string) (*Issued, error) {
	u.Password = ""
	secret, hash := newSecret()
	g := model.Grant{
		UserID:        u.ID,
		Name:          cmp.Or(meta.Name, meta.Client),
		Client:        meta.Client,
		ClientVersion: meta.ClientVersion,
		Scopes:        Entitled(scopes, u.IsAdmin),
		Provider:      provider,
		SecretHash:    hash,
		UserEpoch:     u.TokenEpoch,
		CreatedAt:     s.now(),
	}
	if err := ds.Grant().Put(ctx, &g); err != nil {
		return nil, fmt.Errorf("storing grant: %w", err)
	}
	return &Issued{Secret: secret, Grant: g, User: u}, nil
}

func (s *Service) Authenticate(ctx context.Context, secret, ip string) (*Principal, error) {
	g, err := s.ds.Grant().FindBySecretHash(ctx, hashSecret(secret))
	if errors.Is(err, model.ErrNotFound) {
		return nil, model.ErrInvalidAuth
	}
	if err != nil {
		return nil, err
	}
	if idleSince := s.now().Add(-IdleExpiry); g.LastActivity().Before(idleSince) {
		s.dropIdle(ctx, g.ID, idleSince)
		return nil, model.ErrInvalidAuth
	}
	u, err := s.ds.User().Get(ctx, g.UserID)
	if errors.Is(err, model.ErrNotFound) {
		return nil, model.ErrInvalidAuth
	}
	if err != nil {
		return nil, err
	}
	if g.UserEpoch != u.TokenEpoch {
		if g, u, err = s.settleEpoch(ctx, g.ID); err != nil {
			return nil, err
		}
	}
	s.touch(ctx, g, ip)
	return &Principal{User: *u, GrantID: g.ID, Scopes: Expand(g.Scopes, u.IsAdmin)}, nil
}

// dropIdle deletes only still-idle grants, sparing one renewed meanwhile.
func (s *Service) dropIdle(ctx context.Context, id string, idleSince time.Time) {
	if _, err := s.ds.Grant().DeleteIdle(ctx, idleSince); err != nil {
		log.Warn(ctx, "API v1: could not delete idle grants", "grant", id, err)
	}
}

// settleEpoch re-reads grant and user in one read transaction: separate reads can straddle a password change
// and make a kept grant look dead. Deleting below the snapshot's epoch is safe: a later change only moves kept grants up.
func (s *Service) settleEpoch(ctx context.Context, grantID string) (*model.Grant, *model.User, error) {
	var g *model.Grant
	var u *model.User
	err := s.ds.WithTx(func(tx model.DataStore) error {
		var err error
		if g, err = tx.Grant().Get(ctx, grantID); err != nil {
			return err
		}
		u, err = tx.User().Get(ctx, g.UserID)
		return err
	})
	if errors.Is(err, model.ErrNotFound) {
		return nil, nil, model.ErrInvalidAuth
	}
	if err != nil {
		return nil, nil, err
	}
	if g.UserEpoch != u.TokenEpoch {
		if err := s.ds.Grant().DeleteStaleEpochs(ctx, u.ID, u.TokenEpoch); err != nil {
			log.Warn(ctx, "API v1: could not delete the user's grants from older epochs", "user", u.ID, "grant", grantID, err)
		}
		return nil, nil, model.ErrInvalidAuth
	}
	return g, u, nil
}

// touch writes last_used at most every touchInterval (zero lastUsed: never used); the SQL condition holds that across nodes.
func (s *Service) touch(ctx context.Context, g *model.Grant, ip string) {
	now := s.now()
	if lastUsed := gg.V(g.LastUsedAt); !lastUsed.IsZero() && now.Before(lastUsed.Add(touchInterval)) {
		return
	}
	if err := s.ds.Grant().Touch(ctx, g.ID, ip, now, now.Add(-touchInterval)); err != nil {
		log.Warn(ctx, "API v1: could not record grant use", "grant", g.ID, err)
	}
}

// ListGrants shows only the current epoch: grants left on an older one are dead but only deleted when presented.
func (s *Service) ListGrants(ctx context.Context, p *Principal, offset, limit int) (model.Grants, int64, error) {
	idleSince := s.now().Add(-IdleExpiry)
	grants, err := s.ds.Grant().GetAllForUser(ctx, p.User.ID, p.User.TokenEpoch, idleSince, offset, limit)
	if err != nil {
		return nil, 0, err
	}
	total, err := s.ds.Grant().CountForUser(ctx, p.User.ID, p.User.TokenEpoch, idleSince)
	return grants, total, err
}

func (s *Service) RevokeGrant(ctx context.Context, p *Principal, grantID string) error {
	return s.ds.Grant().DeleteForUser(ctx, p.User.ID, grantID)
}

// Logout succeeds when the grant is already gone, e.g. revoked by another node or a concurrent logout.
func (s *Service) Logout(ctx context.Context, p *Principal) error {
	err := s.RevokeGrant(ctx, p, p.GrantID)
	if errors.Is(err, model.ErrNotFound) {
		return nil
	}
	return err
}

// ChangePassword does every check inside the locked transaction, so a reset that lands first is never overwritten.
func (s *Service) ChangePassword(ctx context.Context, p *Principal, current, newPassword string, revokeOthers bool) error {
	return s.ds.WithTxImmediate(func(tx model.DataStore) error {
		u, err := tx.User().Get(ctx, p.User.ID)
		if errors.Is(err, model.ErrNotFound) {
			return model.ErrInvalidAuth
		}
		if err != nil {
			return err
		}
		g, err := tx.Grant().Get(ctx, p.GrantID)
		if errors.Is(err, model.ErrNotFound) {
			return model.ErrInvalidAuth
		}
		if err != nil {
			return err
		}
		if g.UserID != u.ID || g.UserEpoch != u.TokenEpoch {
			return model.ErrInvalidAuth
		}
		if !PasswordChangeable(*u) {
			return model.ErrNotAuthorized
		}
		res, err := checkCredentials(ctx, s.checkers(tx), u.UserName, current)
		if errors.Is(err, model.ErrInvalidAuth) {
			return ErrCurrentPasswordMismatch
		}
		if err != nil {
			return err
		}
		if !res.PasswordLocal {
			return ErrPasswordManagedExternally
		}
		oldEpoch := u.TokenEpoch
		u.NewPassword = newPassword
		if err := tx.User().Put(ctx, u); err != nil {
			return err
		}
		updated, err := tx.User().Get(ctx, u.ID)
		if err != nil {
			return err
		}
		keep := ""
		if revokeOthers {
			keep = p.GrantID
		}
		if err := tx.Grant().SetEpoch(ctx, u.ID, oldEpoch, updated.TokenEpoch, keep); err != nil {
			return err
		}
		return tx.Grant().DeleteStaleEpochs(ctx, u.ID, updated.TokenEpoch)
	})
}
