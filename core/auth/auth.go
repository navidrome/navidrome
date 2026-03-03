package auth

import (
	"cmp"
	"context"
	"crypto/sha256"
	"sync"
	"time"

	"github.com/go-chi/jwtauth/v5"
	"github.com/lestrrat-go/jwx/v3/jwt"
	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/id"
	"github.com/navidrome/navidrome/model/request"
	"github.com/navidrome/navidrome/utils"
)

var (
	once      sync.Once
	TokenAuth *jwtauth.JWTAuth
)

// Init creates a JWTAuth object from the secret stored in the DB.
// If the secret is not found, it will create a new one and store it in the DB.
func Init(ds model.DataStore) {
	once.Do(func() {
		ctx := context.TODO()
		log.Info("Setting Session Timeout", "value", conf.Server.SessionTimeout)

		secret, err := ds.Property(ctx).Get(consts.JWTSecretKey)
		if err != nil || secret == "" {
			log.Info(ctx, "Creating new JWT secret, used for encrypting UI sessions")
			secret = createNewSecret(ctx, ds)
		} else {
			if secret, err = utils.Decrypt(ctx, getEncKey(), secret); err != nil {
				log.Error(ctx, "Could not decrypt JWT secret, creating a new one", err)
				secret = createNewSecret(ctx, ds)
			}
		}

		TokenAuth = jwtauth.New("HS256", []byte(secret), nil)
	})
}

func CreatePublicToken(claims Claims) (string, error) {
	claims.Issuer = consts.JWTIssuer
	_, token, err := TokenAuth.Encode(claims.ToMap())
	return token, err
}

func CreateExpiringPublicToken(exp time.Time, claims Claims) (string, error) {
	claims.Issuer = consts.JWTIssuer
	if !exp.IsZero() {
		claims.ExpiresAt = exp
	}
	_, token, err := TokenAuth.Encode(claims.ToMap())
	return token, err
}

func CreateToken(u *model.User) (string, error) {
	claims := Claims{
		Issuer:   consts.JWTIssuer,
		Subject:  u.UserName,
		IssuedAt: time.Now(),
		UserID:   u.ID,
		IsAdmin:  u.IsAdmin,
	}
	token, _, err := TokenAuth.Encode(claims.ToMap())
	if err != nil {
		return "", err
	}

	return TouchToken(token)
}

func TouchToken(token jwt.Token) (string, error) {
	claims := ClaimsFromToken(token).
		WithExpiresAt(time.Now().UTC().Add(conf.Server.SessionTimeout))
	_, newToken, err := TokenAuth.Encode(claims.ToMap())
	return newToken, err
}

func Validate(tokenStr string) (Claims, error) {
	token, err := jwtauth.VerifyToken(TokenAuth, tokenStr)
	if err != nil {
		return Claims{}, err
	}
	return ClaimsFromToken(token), nil
}

func WithAdminUser(ctx context.Context, ds model.DataStore) context.Context {
	u, err := ds.User(ctx).FindFirstAdmin()
	if err != nil {
		c, err := ds.User(ctx).CountAll()
		if c == 0 && err == nil {
			log.Debug(ctx, "No admin user yet!", err)
		} else {
			log.Error(ctx, "No admin user found!", err)
		}
		u = &model.User{}
	}

	ctx = request.WithUsername(ctx, u.UserName)
	return request.WithUser(ctx, *u)
}

func createNewSecret(ctx context.Context, ds model.DataStore) string {
	secret := id.NewRandom()
	encSecret, err := utils.Encrypt(ctx, getEncKey(), secret)
	if err != nil {
		log.Error(ctx, "Could not encrypt JWT secret", err)
		return secret
	}
	if err := ds.Property(ctx).Put(consts.JWTSecretKey, encSecret); err != nil {
		log.Error(ctx, "Could not save JWT secret in DB", err)
	}
	return secret
}

func getEncKey() []byte {
	key := cmp.Or(
		conf.Server.PasswordEncryptionKey,
		consts.DefaultEncryptionKey,
	)
	sum := sha256.Sum256([]byte(key))
	return sum[:]
}
