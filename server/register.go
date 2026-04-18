package server

import (
	"encoding/json"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/deluan/rest"
	"github.com/navidrome/navidrome/model/id"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)

type registerRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

func register(ds model.DataStore) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		if !conf.Server.EnableRecommendations {
			_ = rest.RespondWithError(w, http.StatusForbidden, "Registration is disabled")
			return
		}

		var req registerRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			_ = rest.RespondWithError(w, http.StatusUnprocessableEntity, "Invalid request body")
			return
		}

		req.Username = strings.TrimSpace(req.Username)
		req.Email = strings.TrimSpace(strings.ToLower(req.Email))

		// Validate username
		if len(req.Username) < 3 {
			_ = rest.RespondWithError(w, http.StatusBadRequest, "Username must be at least 3 characters")
			return
		}
		if len(req.Username) > 30 {
			_ = rest.RespondWithError(w, http.StatusBadRequest, "Username must be at most 30 characters")
			return
		}

		// Validate email
		if !emailRegex.MatchString(req.Email) {
			_ = rest.RespondWithError(w, http.StatusBadRequest, "Invalid email address")
			return
		}

		// Validate password
		if len(req.Password) < 6 {
			_ = rest.RespondWithError(w, http.StatusBadRequest, "Password must be at least 6 characters")
			return
		}

		// Check if username already exists
		userRepo := ds.User(r.Context())
		_, err := userRepo.FindByUsernameWithPassword(req.Username)
		if err == nil {
			_ = rest.RespondWithError(w, http.StatusConflict, "Username already taken")
			return
		}

		// Create user
		now := time.Now()
		caser := cases.Title(language.Und)
		newUser := model.User{
			ID:          id.NewRandom(),
			UserName:    req.Username,
			Name:        caser.String(req.Username),
			Email:       req.Email,
			NewPassword: req.Password,
			IsAdmin:     false,
			LastLoginAt: &now,
		}

		err = userRepo.Put(&newUser)
		if err != nil {
			log.Error(r, "Failed to create user", "username", req.Username, err)
			_ = rest.RespondWithError(w, http.StatusInternalServerError, "Failed to create account")
			return
		}

		log.Info(r, "New user registered", "username", req.Username, "email", req.Email)

		// Auto-login after registration
		doLogin(ds, req.Username, req.Password, w, r)
	}
}
