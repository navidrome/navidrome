package app

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/cloudsonic/sonic-server/consts"
	"github.com/cloudsonic/sonic-server/model"
	"github.com/deluan/rest"
	"github.com/dgrijalva/jwt-go"
	"github.com/go-chi/jwtauth"
	log "github.com/sirupsen/logrus"
)

var (
	once      sync.Once
	jwtSecret []byte
	TokenAuth *jwtauth.JWTAuth
)

func Login(ds model.DataStore) func(w http.ResponseWriter, r *http.Request) {
	initTokenAuth(ds)

	return func(w http.ResponseWriter, r *http.Request) {
		data := make(map[string]string)
		decoder := json.NewDecoder(r.Body)
		if err := decoder.Decode(&data); err != nil {
			log.Errorf("parsing request body: %#v", err)
			rest.RespondWithError(w, http.StatusUnprocessableEntity, "Invalid request payload")
			return
		}
		username := data["username"]
		password := data["password"]

		user, err := validateLogin(ds.User(), username, password)
		if err != nil {
			rest.RespondWithError(w, http.StatusInternalServerError, "Unknown error authentication user. Please try again")
			return
		}
		if user == nil {
			log.Warnf("Unsuccessful login: '%s', request: %v", username, r.Header)
			rest.RespondWithError(w, http.StatusUnauthorized, "Invalid username or password")
			return
		}

		tokenString, err := createToken(user)
		if err != nil {
			rest.RespondWithError(w, http.StatusInternalServerError, "Unknown error authenticating user. Please try again")
		}
		rest.RespondWithJSON(w, http.StatusOK,
			map[string]interface{}{
				"message":  "User '" + username + "' authenticated successfully",
				"token":    tokenString,
				"name":     strings.Title(user.Name),
				"username": username,
			})
	}
}

func initTokenAuth(ds model.DataStore) {
	once.Do(func() {
		secret, err := ds.Property().DefaultGet(consts.JWTSecretKey, "not so secret")
		if err != nil {
			log.Error("No JWT secret found in DB. Setting a temp one, but please report this error", err)
		}
		jwtSecret = []byte(secret)
		TokenAuth = jwtauth.New("HS256", jwtSecret, nil)
	})
}
func validateLogin(userRepo model.UserRepository, userName, password string) (*model.User, error) {
	u, err := userRepo.FindByUsername(userName)
	if err == model.ErrNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if u.Password != password {
		return nil, nil
	}
	if !u.IsAdmin {
		log.Warn("Non-admin user tried to login", "user", userName)
		return nil, nil
	}
	err = userRepo.UpdateLastLoginAt(u.ID)
	if err != nil {
		log.Error("Could not update LastLoginAt", "user", userName)
	}
	return u, nil
}

func createToken(u *model.User) (string, error) {
	token := jwt.New(jwt.SigningMethodHS256)
	claims := token.Claims.(jwt.MapClaims)
	claims["iss"] = consts.JWTIssuer
	claims["sub"] = u.UserName

	return touchToken(token)
}

func touchToken(token *jwt.Token) (string, error) {
	expireIn := time.Now().Add(consts.JWTTokenExpiration).Unix()
	claims := token.Claims.(jwt.MapClaims)
	claims["exp"] = expireIn

	return token.SignedString(jwtSecret)
}

func userFrom(claims jwt.MapClaims) *model.User {
	user := &model.User{
		UserName: claims["sub"].(string),
	}
	return user
}

func Authenticator(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token, _, err := jwtauth.FromContext(r.Context())

		if err != nil {
			rest.RespondWithError(w, http.StatusUnauthorized, "Not authenticated")
			return
		}

		if token == nil || !token.Valid {
			rest.RespondWithError(w, http.StatusUnauthorized, "Not authenticated")
			return
		}

		claims := token.Claims.(jwt.MapClaims)

		newCtx := context.WithValue(r.Context(), "loggedUser", userFrom(claims))
		newTokenString, err := touchToken(token)
		if err != nil {
			log.Errorf("signing new token: %v", err)
			rest.RespondWithError(w, http.StatusUnauthorized, "Not authenticated")
			return
		}

		w.Header().Set("Authorization", newTokenString)
		next.ServeHTTP(w, r.WithContext(newCtx))
	})
}
