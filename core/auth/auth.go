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
	once sync.Once
	// TokenAuth signs UI/API session tokens. Rotated by the id migration so stale sessions die.
	TokenAuth *jwtauth.JWTAuth
	// PublicTokenAuth signs public-link tokens (artwork, share streams), on a separate secret that survives the rotation.
	PublicTokenAuth *jwtauth.JWTAuth
)

// Init creates the JWTAuth objects from the secrets stored in the DB.
// Missing or undecryptable secrets are regenerated and stored.
func Init(ds model.DataStore) {
	once.Do(func() {
		ctx := context.TODO()
		log.Info("Setting Session Timeout", "value", conf.Server.SessionTimeout)

		TokenAuth = jwtauth.New("HS256", []byte(loadOrCreateSecret(ctx, ds, consts.JWTSecretKey)), nil)
		PublicTokenAuth = jwtauth.New("HS256", []byte(loadOrCreateSecret(ctx, ds, consts.JWTPublicSecretKey)), nil)
	})
}

func loadOrCreateSecret(ctx context.Context, ds model.DataStore, key string) string {
	secret, err := ds.Property(ctx).Get(key)
	if err != nil || secret == "" {
		log.Info(ctx, "Creating new JWT secret", "key", key)
		return createNewSecret(ctx, ds, key)
	}
	if secret, err = utils.Decrypt(ctx, getEncKey(), secret); err != nil {
		log.Error(ctx, "Could not decrypt JWT secret, creating a new one", "key", key, err)
		return createNewSecret(ctx, ds, key)
	}
	return secret
}

func CreatePublicToken(claims Claims) (string, error) {
	claims.Issuer = consts.JWTIssuer
	_, token, err := PublicTokenAuth.Encode(claims.ToMap())
	return token, err
}

func CreateExpiringPublicToken(exp time.Time, claims Claims) (string, error) {
	claims.Issuer = consts.JWTIssuer
	if !exp.IsZero() {
		claims.ExpiresAt = exp
	}
	_, token, err := PublicTokenAuth.Encode(claims.ToMap())
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

// ValidatePublic verifies a public-link token against the public secret.
func ValidatePublic(tokenStr string) (Claims, error) {
	token, err := jwtauth.VerifyToken(PublicTokenAuth, tokenStr)
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
		u = &model.User{IsAdmin: true, UserName: "admin"}
	}

	ctx = request.WithUsername(ctx, u.UserName)
	return request.WithUser(ctx, *u)
}

func createNewSecret(ctx context.Context, ds model.DataStore, key string) string {
	secret := id.NewRandom()
	encSecret, err := utils.Encrypt(ctx, getEncKey(), secret)
	if err != nil {
		log.Error(ctx, "Could not encrypt JWT secret", err)
		return secret
	}
	if err := ds.Property(ctx).Put(key, encSecret); err != nil {
		log.Error(ctx, "Could not save JWT secret in DB", err)
	}
	return secret
}

// EncodeToken creates a signed JWT from an arbitrary claims map.
// It sets the issuer claim automatically.
func EncodeToken(claims map[string]any) (string, error) {
	claims[jwt.IssuerKey] = consts.JWTIssuer
	_, token, err := TokenAuth.Encode(claims)
	return token, err
}

// DecodeAndVerifyToken verifies a JWT string and returns the parsed token.
func DecodeAndVerifyToken(tokenStr string) (jwt.Token, error) {
	return jwtauth.VerifyToken(TokenAuth, tokenStr)
}

func getEncKey() []byte {
	key := cmp.Or(
		conf.Server.PasswordEncryptionKey,
		consts.DefaultEncryptionKey,
	)
	sum := sha256.Sum256([]byte(key))
	return sum[:]
}
