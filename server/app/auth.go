package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/deluan/navidrome/conf"
	"github.com/deluan/navidrome/consts"
	"github.com/deluan/navidrome/core/auth"
	"github.com/deluan/navidrome/log"
	"github.com/deluan/navidrome/model"
	"github.com/deluan/navidrome/model/request"
	"github.com/deluan/rest"
	"github.com/dgrijalva/jwt-go"
	"github.com/go-chi/jwtauth"
	"github.com/go-ldap/ldap"
	"github.com/google/uuid"
)

var (
	ErrFirstTime = errors.New("no users created")
)

func Login(ds model.DataStore) func(w http.ResponseWriter, r *http.Request) {
	auth.InitTokenAuth(ds)

	return func(w http.ResponseWriter, r *http.Request) {
		username, password, err := getCredentialsFromBody(r)
		if err != nil {
			log.Error(r, "Parsing request body", err)
			_ = rest.RespondWithError(w, http.StatusUnprocessableEntity, err.Error())
			return
		}

		handleLogin(ds, username, password, w, r)
	}
}

func handleLogin(ds model.DataStore, username string, password string, w http.ResponseWriter, r *http.Request) {
	user, err := validateLogin(ds.User(r.Context()), username, password)
	if err != nil {
		_ = rest.RespondWithError(w, http.StatusInternalServerError, "Unknown error authentication user. Please try again")
		return
	}
	if user == nil {
		log.Warn(r, "Unsuccessful login", "username", username, "request", r.Header)
		_ = rest.RespondWithError(w, http.StatusUnauthorized, "Invalid username or password")
		return
	}

	tokenString, err := auth.CreateToken(user)
	if err != nil {
		_ = rest.RespondWithError(w, http.StatusInternalServerError, "Unknown error authenticating user. Please try again")
		return
	}
	_ = rest.RespondWithJSON(w, http.StatusOK,
		map[string]interface{}{
			"message":  "User '" + username + "' authenticated successfully",
			"token":    tokenString,
			"name":     user.Name,
			"username": username,
			"isAdmin":  user.IsAdmin,
		})
}

func getCredentialsFromBody(r *http.Request) (username string, password string, err error) {
	data := make(map[string]string)
	decoder := json.NewDecoder(r.Body)
	if err = decoder.Decode(&data); err != nil {
		log.Error(r, "parsing request body", err)
		err = errors.New("invalid request payload")
		return
	}
	username = data["username"]
	password = data["password"]
	return username, password, nil
}

func CreateAdmin(ds model.DataStore) func(w http.ResponseWriter, r *http.Request) {
	auth.InitTokenAuth(ds)

	return func(w http.ResponseWriter, r *http.Request) {
		username, password, err := getCredentialsFromBody(r)
		if err != nil {
			log.Error(r, "parsing request body", err)
			_ = rest.RespondWithError(w, http.StatusUnprocessableEntity, err.Error())
			return
		}
		c, err := ds.User(r.Context()).CountAll()
		if err != nil {
			_ = rest.RespondWithError(w, http.StatusInternalServerError, err.Error())
			return
		}
		if c > 0 {
			_ = rest.RespondWithError(w, http.StatusForbidden, "Cannot create another first admin")
			return
		}
		err = createDefaultUser(r.Context(), ds, username, password)
		if err != nil {
			_ = rest.RespondWithError(w, http.StatusInternalServerError, err.Error())
			return
		}
		handleLogin(ds, username, password, w, r)
	}
}

func createDefaultUser(ctx context.Context, ds model.DataStore, username, password string) error {
	id, _ := uuid.NewRandom()
	log.Warn("Creating initial user", "user", username)
	now := time.Now()
	initialUser := model.User{
		ID:          id.String(),
		UserName:    username,
		Name:        strings.Title(username),
		Email:       "",
		Password:    password,
		IsAdmin:     true,
		LastLoginAt: &now,
	}
	err := ds.User(ctx).Put(&initialUser)
	if err != nil {
		log.Error("Could not create initial user", "user", initialUser, err)
	}
	return nil
}

func validateLogin(userRepo model.UserRepository, userName, password string) (*model.User, error) {
	u, err := validateLoginLDAP(userRepo, userName, password)
	if u != nil && err == nil {
		return u, nil
	}
	u, err = userRepo.FindByUsername(userName)
	if err == model.ErrNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if u.Password != password {
		return nil, nil
	}
	err = userRepo.UpdateLastLoginAt(u.ID)
	if err != nil {
		log.Error("Could not update LastLoginAt", "user", userName)
	}
	return u, nil
}

func validateLoginLDAP(userRepo model.UserRepository, userName, password string) (*model.User, error) {
	binduserdn := conf.Server.LDAP.BindDN
	bindpassword := conf.Server.LDAP.BindPassword
	mailAttr := conf.Server.LDAP.Mail
	nameAttr := conf.Server.LDAP.Name

	l, err := ldap.DialURL(conf.Server.LDAP.Host)
	if err != nil {
		log.Error(err)
	}
	defer l.Close()

	// First bind with a read only user
	err = l.Bind(binduserdn, bindpassword)
	if err != nil {
		log.Error(err)
	}

	// Search for the given username
	searchRequest := ldap.NewSearchRequest(
		conf.Server.LDAP.Base,
		ldap.ScopeWholeSubtree, ldap.NeverDerefAliases, 0, 0, false,
		fmt.Sprintf(conf.Server.LDAP.SearchFilter, ldap.EscapeFilter(userName)),
		[]string{"dn", nameAttr, mailAttr},
		nil,
	)

	sr, err := l.Search(searchRequest)
	if err != nil {
		log.Error(err)
	}

	if len(sr.Entries) != 1 {
		log.Error("User does not exist or too many entries returned")
		return nil, nil
	}

	dn := sr.Entries[0].DN
	mail := sr.Entries[0].GetAttributeValue(mailAttr)
	name := sr.Entries[0].GetAttributeValue(nameAttr)

	authenticated := true
	// Bind as the user to verify their password
	err = l.Bind(dn, password)

	if err != nil {
		log.Error(err)
		authenticated = false
	}

	// Rebind as the read only user for any further queries
	err = l.Bind(binduserdn, bindpassword)
	if err != nil {
		log.Error(err)
	}

	if !authenticated {
		return nil, nil
	}

	u, err := userRepo.FindByUsername(userName)
	if err == model.ErrNotFound {
		u = &model.User{UserName: userName}
	}
	u.Name = name
	u.Email = mail
	u.Password = password
	err = userRepo.Put(u)
	if err != nil {
		log.Error("Could not update User", "user", userName)
	}

	err = userRepo.UpdateLastLoginAt(u.ID)
	if err != nil {
		log.Error("Could not update LastLoginAt", "user", userName)
	}

	return u, nil
}

func contextWithUser(ctx context.Context, ds model.DataStore, claims jwt.MapClaims) context.Context {
	userName := claims["sub"].(string)
	user, _ := ds.User(ctx).FindByUsername(userName)
	return request.WithUser(ctx, *user)
}

func getToken(ds model.DataStore, ctx context.Context) (*jwt.Token, error) {
	token, claims, err := jwtauth.FromContext(ctx)

	valid := err == nil && token != nil && token.Valid
	valid = valid && claims["sub"] != nil
	if valid {
		return token, nil
	}

	c, err := ds.User(ctx).CountAll()
	firstTime := c == 0 && err == nil
	if firstTime {
		return nil, ErrFirstTime
	}
	return nil, errors.New("invalid authentication")
}

// This method maps the custom authorization header to the default 'Authorization', used by the jwtauth library
func mapAuthHeader() func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			bearer := r.Header.Get(consts.UIAuthorizationHeader)
			r.Header.Set("Authorization", bearer)
			next.ServeHTTP(w, r)
		})
	}
}

func authenticator(ds model.DataStore) func(next http.Handler) http.Handler {
	auth.InitTokenAuth(ds)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token, err := getToken(ds, r.Context())
			if err == ErrFirstTime {
				_ = rest.RespondWithJSON(w, http.StatusUnauthorized, map[string]string{"message": ErrFirstTime.Error()})
				return
			}
			if err != nil {
				_ = rest.RespondWithError(w, http.StatusUnauthorized, "Not authenticated")
				return
			}

			claims := token.Claims.(jwt.MapClaims)

			newCtx := contextWithUser(r.Context(), ds, claims)
			newTokenString, err := auth.TouchToken(token)
			if err != nil {
				log.Error(r, "signing new token", err)
				_ = rest.RespondWithError(w, http.StatusUnauthorized, "Not authenticated")
				return
			}

			w.Header().Set(consts.UIAuthorizationHeader, newTokenString)
			next.ServeHTTP(w, r.WithContext(newCtx))
		})
	}
}
