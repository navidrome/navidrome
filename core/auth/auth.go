package auth

import (
	"context"
	"sync"
	"time"

	"github.com/go-chi/jwtauth/v5"
	"github.com/lestrrat-go/jwx/jwt"
	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
)

var (
	once           sync.Once
	Secret         []byte
	TokenAuth      *jwtauth.JWTAuth
	sessionTimeOut time.Duration
)

func InitTokenAuth(ds model.DataStore) {
	once.Do(func() {
		secret, err := ds.Property(context.TODO()).DefaultGet(consts.JWTSecretKey, "not so secret")
		if err != nil {
			log.Error("No JWT secret found in DB. Setting a temp one, but please report this error", err)
		}
		Secret = []byte(secret)
		TokenAuth = jwtauth.New("HS256", Secret, nil)
	})
}

func CreateToken(u *model.User) (string, error) {
	claims := map[string]interface{}{}
	claims[jwt.IssuerKey] = consts.JWTIssuer
	claims[jwt.IssuedAtKey] = time.Now().UTC().Unix()
	claims[jwt.SubjectKey] = u.UserName
	claims["uid"] = u.ID
	claims["adm"] = u.IsAdmin
	token, _, err := TokenAuth.Encode(claims)
	if err != nil {
		return "", err
	}

	return TouchToken(token)
}

func getSessionTimeOut() time.Duration {
	if sessionTimeOut == 0 {
		sessionTimeOut = conf.Server.SessionTimeout
		log.Info("Setting Session Timeout", "value", sessionTimeOut)
	}
	return sessionTimeOut
}

func TouchToken(token jwt.Token) (string, error) {
	claims, err := token.AsMap(context.Background())
	if err != nil {
		return "", err
	}

	timeout := getSessionTimeOut()
	claims[jwt.ExpirationKey] = time.Now().UTC().Add(timeout).Unix()
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
