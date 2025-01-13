package auth

import (
	"cmp"
	"context"
	"crypto/sha256"
	"sync"
	"time"

	"github.com/go-chi/jwtauth/v5"
	"github.com/lestrrat-go/jwx/v2/jwt"
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

func createBaseClaims() map[string]any {
	tokenClaims := map[string]any{}
	tokenClaims[jwt.IssuerKey] = consts.JWTIssuer
	return tokenClaims
}

func CreatePublicToken(claims map[string]any) (string, error) {
	tokenClaims := createBaseClaims()
	for k, v := range claims {
		tokenClaims[k] = v
	}
	_, token, err := TokenAuth.Encode(tokenClaims)

	return token, err
}

func CreateExpiringPublicToken(exp time.Time, claims map[string]any) (string, error) {
	tokenClaims := createBaseClaims()
	if !exp.IsZero() {
		tokenClaims[jwt.ExpirationKey] = exp.UTC().Unix()
	}
	for k, v := range claims {
		tokenClaims[k] = v
	}
	_, token, err := TokenAuth.Encode(tokenClaims)

	return token, err
}

func CreateToken(u *model.User) (string, error) {
	claims := createBaseClaims()
	claims[jwt.SubjectKey] = u.UserName
	claims[jwt.IssuedAtKey] = time.Now().UTC().Unix()
	claims["uid"] = u.ID
	claims["adm"] = u.IsAdmin
	token, _, err := TokenAuth.Encode(claims)
	if err != nil {
		return "", err
	}

	return TouchToken(token)
}

func TouchToken(token jwt.Token) (string, error) {
	claims, err := token.AsMap(context.Background())
	if err != nil {
		return "", err
	}

	claims[jwt.ExpirationKey] = time.Now().UTC().Add(conf.Server.SessionTimeout).Unix()
	_, newToken, err := TokenAuth.Encode(claims)

	return newToken, err
}

func Validate(tokenStr string) (map[string]interface{}, error) {
	token, err := jwtauth.VerifyToken(TokenAuth, tokenStr)
	if err != nil {
		return nil, err
	}
	return token.AsMap(context.Background())
}

func WithAdminUser(ctx context.Context, ds model.DataStore) context.Context {
	u, err := ds.User(ctx).FindFirstAdmin()
	if err != nil {
		c, err := ds.User(ctx).CountAll()
		if c == 0 && err == nil {
			log.Debug(ctx, "Scanner: No admin user yet!", err)
		} else {
			log.Error(ctx, "Scanner: No admin user found!", err)
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
