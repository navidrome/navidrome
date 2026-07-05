package jellyfin

import (
	"encoding/json"
	"net/http"

	"github.com/navidrome/navidrome/core/auth"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/server/jellyfin/dto"
)

type authenticateByNameRequest struct {
	Username string `json:"Username"`
	Pw       string `json:"Pw"`
}

func (api *Router) authenticateByName(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var body authenticateByNameRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	// Navidrome stores recoverable passwords; this mirrors Subsonic's validateCredentials plaintext path.
	usr, err := api.ds.User(ctx).FindByUsernameWithPassword(body.Username)
	if err != nil || usr.Password != body.Pw {
		log.Warn(ctx, "Jellyfin API: invalid login", "username", body.Username, "remoteAddr", r.RemoteAddr)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	token, err := auth.CreateToken(usr)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	api.ok(w, r, dto.AuthenticationResult{
		User:        userToDto(usr, api.serverName()),
		AccessToken: token,
		ServerId:    api.publicInfo(ctx).Id,
		SessionInfo: &dto.SessionInfo{Id: parseEmbyAuth(r).DeviceId, UserId: usr.ID},
	})
}

func userToDto(u *model.User, serverName string) *dto.UserDto {
	return &dto.UserDto{
		Name:                  u.UserName,
		Id:                    u.ID,
		HasPassword:           true,
		HasConfiguredPassword: true,
	}
}
