package apiauth

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"slices"
	"sync"
	"sync/atomic"
	"time"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/core/auth"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/utils/gg"
)

const (
	TokenTTL      = time.Hour
	IdleExpiry    = consts.APIv1GrantIdleExpiry
	cacheTTL      = 30 * time.Second
	touchInterval = 5 * time.Minute
)

var (
	ErrInsufficientScope         = errors.New("insufficient scope")
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

type AccessToken struct {
	Token     string
	ExpiresIn time.Duration
	Scopes    []string
}

type Principal struct {
	User    model.User
	GrantID string
	Scopes  []string
}

type Service struct {
	ds       model.DataStore
	checkers func(ds model.DataStore) []CredentialChecker // per datastore, so password change can check inside its transaction
	cache    *livenessCache
	now      func() time.Time
	signerMu sync.Mutex
	sg       atomic.Pointer[signer]
}

func New(ds model.DataStore) *Service {
	s := &Service{
		ds: ds,
		checkers: func(ds model.DataStore) []CredentialChecker {
			return []CredentialChecker{dbChecker{ds: ds}}
		},
		cache: newLivenessCache(cacheTTL),
		now:   time.Now,
	}
	return s
}

// signer loads the key on first use, so building the router never touches the database; only a success is kept.
func (s *Service) signer() (*signer, error) {
	if sg := s.sg.Load(); sg != nil {
		return sg, nil
	}
	s.signerMu.Lock()
	defer s.signerMu.Unlock()
	if sg := s.sg.Load(); sg != nil {
		return sg, nil
	}
	sg, err := loadSigner(context.Background(), s.ds, func() time.Time { return s.now() })
	if err != nil {
		return nil, err
	}
	s.sg.Store(sg)
	return sg, nil
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

func (s *Service) ResolveGrant(ctx context.Context, secret, ip string) (*Principal, error) {
	g, err := s.ds.Grant().FindBySecretHash(ctx, hashSecret(secret))
	if errors.Is(err, model.ErrNotFound) {
		return nil, model.ErrInvalidAuth
	}
	if err != nil {
		return nil, err
	}
	if !s.now().Before(g.LastActivity().Add(IdleExpiry)) {
		s.dropGrant(ctx, g.ID)
		return nil, model.ErrInvalidAuth
	}
	u, err := s.loadUser(ctx, g.UserID)
	if err != nil {
		return nil, err
	}
	if g.UserEpoch != u.TokenEpoch {
		if g, u, err = s.settleEpoch(ctx, g.ID); err != nil {
			return nil, err
		}
	}
	s.touch(ctx, g.ID, ip, gg.V(g.LastUsedAt))
	return &Principal{User: *u, GrantID: g.ID, Scopes: Expand(g.Scopes, u.IsAdmin)}, nil
}

func (s *Service) Mint(ctx context.Context, p *Principal, requested []string) (*AccessToken, error) {
	sg, err := s.signer()
	if err != nil {
		return nil, err
	}
	now := s.now()
	scopes := Attenuate(p.Scopes, requested)
	tok, err := sg.sign(claims{UserID: p.User.ID, GrantID: p.GrantID, Scopes: scopes, IssuedAt: now, ExpiresAt: now.Add(TokenTTL)})
	if err != nil {
		return nil, fmt.Errorf("signing access token: %w", err)
	}
	return &AccessToken{Token: tok, ExpiresIn: TokenTTL, Scopes: scopes}, nil
}

func (s *Service) loadUser(ctx context.Context, userID string) (*model.User, error) {
	u, err := s.ds.User().Get(ctx, userID)
	if errors.Is(err, model.ErrNotFound) {
		return nil, model.ErrInvalidAuth
	}
	return u, err
}

// dropGrant evicts after deleting, so a concurrent fill cannot re-cache the dead grant.
func (s *Service) dropGrant(ctx context.Context, id string) {
	if err := s.ds.Grant().Delete(ctx, id); err != nil {
		log.Warn(ctx, "API v1: could not delete dead grant", "grant", id, err)
	}
	s.cache.evict(id)
}

// settleEpoch re-reads grant and user in one read transaction: separate reads can straddle a password
// change and make a kept grant look dead. The delete only fires while the grant is on the epoch seen here.
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
		s.cache.evict(grantID)
		return nil, nil, model.ErrInvalidAuth
	}
	if err != nil {
		return nil, nil, err
	}
	if g.UserEpoch != u.TokenEpoch {
		s.cache.evict(grantID)
		if err := s.ds.Grant().DeleteIfEpoch(ctx, grantID, g.UserEpoch); err != nil {
			log.Warn(ctx, "API v1: could not delete dead grant", "grant", grantID, err)
		}
		return nil, nil, model.ErrInvalidAuth
	}
	return g, u, nil
}

// touch writes last_used at most every touchInterval (a zero lastUsed means never); the SQL condition keeps that true across nodes.
func (s *Service) touch(ctx context.Context, id, ip string, lastUsed time.Time) {
	now := s.now()
	if !lastUsed.IsZero() && now.Before(lastUsed.Add(touchInterval)) {
		return
	}
	if err := s.ds.Grant().Touch(ctx, id, ip, now, now.Add(-touchInterval)); err != nil {
		log.Warn(ctx, "API v1: could not record grant use", "grant", id, err)
		return
	}
	s.cache.markUsed(id, now)
}

func (s *Service) Authenticate(ctx context.Context, token, ip string) (*Principal, error) {
	sg, err := s.signer()
	if err != nil {
		return nil, err
	}
	c, err := sg.parse(token)
	if err != nil {
		return nil, err
	}
	u, err := s.loadUser(ctx, c.UserID)
	if err != nil {
		return nil, err
	}
	entry, u, err := s.liveGrant(ctx, c.GrantID, u)
	if err != nil {
		return nil, err
	}
	if entry.userID != c.UserID {
		return nil, model.ErrInvalidAuth
	}
	if slices.Contains(c.Scopes, ScopeAdmin) && !u.IsAdmin {
		return nil, ErrInsufficientScope
	}
	s.touch(ctx, c.GrantID, ip, entry.lastUsedAt)
	return &Principal{User: *u, GrantID: c.GrantID, Scopes: Allowed(c.Scopes, u.IsAdmin)}, nil
}

// liveGrant trusts the cache only while its epoch matches; a mismatch is settled from one consistent read.
func (s *Service) liveGrant(ctx context.Context, id string, u *model.User) (livenessEntry, *model.User, error) {
	now := s.now()
	if e, ok := s.cache.get(id, now); ok && e.epoch == u.TokenEpoch {
		return e, u, nil
	}
	started := s.cache.begin()
	g, err := s.ds.Grant().Get(ctx, id)
	if errors.Is(err, model.ErrNotFound) {
		s.cache.evict(id)
		return livenessEntry{}, nil, model.ErrInvalidAuth
	}
	if err != nil {
		return livenessEntry{}, nil, err
	}
	if g.UserEpoch != u.TokenEpoch {
		if g, u, err = s.settleEpoch(ctx, id); err != nil {
			return livenessEntry{}, nil, err
		}
	}
	e := livenessEntry{userID: g.UserID, epoch: g.UserEpoch, lastUsedAt: gg.V(g.LastUsedAt)}
	s.cache.put(id, e, now, started)
	return e, u, nil
}

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
	if err := s.ds.Grant().DeleteForUser(ctx, p.User.ID, grantID); err != nil {
		return err
	}
	s.cache.evict(grantID)
	return nil
}

// Logout succeeds when the grant is already gone, e.g. revoked by another node or a concurrent logout.
func (s *Service) Logout(ctx context.Context, p *Principal) error {
	err := s.RevokeGrant(ctx, p, p.GrantID)
	if errors.Is(err, model.ErrNotFound) {
		s.cache.evict(p.GrantID)
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
		return tx.Grant().DeleteOtherEpochs(ctx, u.ID, updated.TokenEpoch)
	})
}
