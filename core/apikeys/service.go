package apikeys

import (
	"cmp"
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/core/apikeyworker"
	"github.com/navidrome/navidrome/core/auth"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/id"
	"github.com/navidrome/navidrome/utils"
)

const tokenPrefix = "nd_"

type Service struct {
	ds model.DataStore
}

func New(ds model.DataStore) *Service {
	return &Service{ds: ds}
}

type CreateInput struct {
	Name      string
	ExpiresAt *time.Time
}

func (s *Service) Create(ctx context.Context, userID string, input CreateInput) (*model.UserAPIKey, string, error) {
	name := strings.TrimSpace(input.Name)
	if name == "" {
		return nil, "", errors.New("name is required")
	}
	now := time.Now()
	if input.ExpiresAt != nil && input.ExpiresAt.Before(now) {
		return nil, "", errors.New("expiresAt must be in the future")
	}
	pepper, err := s.pepper(ctx)
	if err != nil {
		return nil, "", err
	}

	token, lookupPrefix, hash, err := s.generateToken(ctx, pepper)
	if err != nil {
		return nil, "", err
	}

	key := &model.UserAPIKey{
		ID:           id.NewRandom(),
		UserID:       userID,
		Name:         name,
		LookupPrefix: lookupPrefix,
		TokenHash:    hash,
		ExpiresAt:    input.ExpiresAt,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if err := s.ds.APIKey(ctx).Put(key); err != nil {
		return nil, "", err
	}
	return key, token, nil
}

func (s *Service) List(ctx context.Context, userID string) (model.UserAPIKeys, error) {
	return s.ds.APIKey(ctx).GetByUserID(userID)
}

func (s *Service) Delete(ctx context.Context, userID, keyID string) error {
	key, err := s.ds.APIKey(ctx).Get(keyID)
	if err != nil {
		return err
	}
	if key.UserID != userID {
		return model.ErrNotAuthorized
	}
	return s.ds.APIKey(ctx).Delete(keyID)
}

func (s *Service) Authenticate(ctx context.Context, token string) (*model.User, error) {
	if strings.HasPrefix(token, tokenPrefix) {
		return s.authenticateDedicatedKey(ctx, token)
	}
	claims, err := auth.Validate(token)
	if err != nil {
		return nil, err
	}
	return s.ds.User(ctx).Get(claims.UserID)
}

func (s *Service) authenticateDedicatedKey(ctx context.Context, token string) (*model.User, error) {
	lookupPrefix := dedicatedLookupPrefix(token)
	if lookupPrefix == "" {
		return nil, model.ErrInvalidAuth
	}
	keys, err := s.ds.APIKey(ctx).FindByLookupPrefix(lookupPrefix)
	if err != nil || len(keys) == 0 {
		return nil, model.ErrInvalidAuth
	}
	pepper, err := s.pepper(ctx)
	if err != nil {
		return nil, err
	}
	for i := range keys {
		candidate := keys[i]
		if candidate.ExpiresAt != nil && candidate.ExpiresAt.Before(time.Now()) {
			continue
		}
		valid, err := s.verifyToken(ctx, token, candidate.TokenHash, pepper)
		if err != nil || !valid {
			continue
		}
		_ = s.ds.APIKey(ctx).TouchLastUsed(candidate.ID, time.Now())
		return s.ds.User(ctx).Get(candidate.UserID)
	}
	return nil, model.ErrInvalidAuth
}

func (s *Service) generateToken(ctx context.Context, pepper string) (token, lookupPrefix, hash string, err error) {
	result, workerErr := apikeyworker.Generate(ctx, pepper)
	if workerErr == nil {
		return result.Token, result.LookupPrefix, result.Hash, nil
	}
	return "", "", "", fmt.Errorf("api key hashing worker unavailable: %w", workerErr)
}

func (s *Service) verifyToken(ctx context.Context, token, hash, pepper string) (bool, error) {
	valid, workerErr := apikeyworker.Verify(ctx, token, hash, pepper)
	if workerErr == nil {
		return valid, nil
	}
	return false, fmt.Errorf("api key hashing worker unavailable: %w", workerErr)
}

func (s *Service) pepper(ctx context.Context) (string, error) {
	auth.Init(s.ds)
	secret, err := s.ds.Property(ctx).Get(consts.JWTSecretKey)
	if err != nil || secret == "" {
		return "", fmt.Errorf("api key pepper unavailable: %w", err)
	}
	plain, err := utils.Decrypt(ctx, authEncKey(), secret)
	if err != nil {
		return "", fmt.Errorf("decrypt api key pepper: %w", err)
	}
	if len(plain) < 16 {
		return "", errors.New("api key pepper is too short")
	}
	return plain, nil
}

func authEncKey() []byte {
	key := cmp.Or(conf.Server.PasswordEncryptionKey, consts.DefaultEncryptionKey)
	sum := sha256.Sum256([]byte(key))
	return sum[:]
}

func dedicatedLookupPrefix(token string) string {
	body := strings.TrimPrefix(token, tokenPrefix)
	if body == "" {
		return ""
	}
	runes := []rune(body)
	if len(runes) < 8 {
		return ""
	}
	if len(runes) > 12 {
		runes = runes[:12]
	}
	return string(runes)
}

func generateTokenGo(pepper string) (token, lookupPrefix, hash string, err error) {
	raw := id.NewRandom()
	token = tokenPrefix + raw
	lookupPrefix = dedicatedLookupPrefix(token)
	hash = hashTokenGo(token, pepper)
	return token, lookupPrefix, hash, nil
}

func hashTokenGo(token, pepper string) string {
	sum := sha256.Sum256([]byte(pepper + token))
	return hex.EncodeToString(sum[:])
}

func verifyTokenGo(token, expectedHash, pepper string) bool {
	actual := hashTokenGo(token, pepper)
	return subtle.ConstantTimeCompare([]byte(actual), []byte(expectedHash)) == 1
}
