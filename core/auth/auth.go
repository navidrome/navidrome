package auth

import (
	"context"
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
)

var (
	once      sync.Once
	Secret    []byte
	TokenAuth *jwtauth.JWTAuth
)

func Init(ds model.DataStore) {
	once.Do(func() {
		log.Info("Setting Session Timeout", "value", conf.Server.SessionTimeout)
		secret, err := ds.Property(context.TODO()).Get(consts.JWTSecretKey)
		if err != nil || secret == "" {
			log.Error("No JWT secret found in DB. Setting a temp one, but please report this error", err)
			secret = id.NewRandom()
		}
		Secret = []byte(secret)
		TokenAuth = jwtauth.New("HS256", Secret, nil)
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
