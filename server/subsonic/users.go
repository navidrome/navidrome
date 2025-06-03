package subsonic

import (
	"net/http"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
	"github.com/navidrome/navidrome/server/subsonic/responses"
)

// buildUserResponse creates a User response object from a User model
func buildUserResponse(user model.User) responses.User {
	userResponse := responses.User{
		Username:          user.UserName,
		AdminRole:         user.IsAdmin,
		Email:             user.Email,
		StreamRole:        true,
		ScrobblingEnabled: true,
		DownloadRole:      conf.Server.EnableDownloads,
		ShareRole:         conf.Server.EnableSharing,
	}

	if conf.Server.Jukebox.Enabled {
		userResponse.JukeboxRole = !conf.Server.Jukebox.AdminOnly || user.IsAdmin
	}

	return userResponse
}

// TODO This is a placeholder. The real one has to read this info from a config file or the database
func (api *Router) GetUser(r *http.Request) (*responses.Subsonic, error) {
	loggedUser, ok := request.UserFrom(r.Context())
	if !ok {
		return nil, newError(responses.ErrorGeneric, "Internal error")
	}

	response := newResponse()
	user := buildUserResponse(loggedUser)
	response.User = &user
	return response, nil
}

func (api *Router) GetUsers(r *http.Request) (*responses.Subsonic, error) {
	loggedUser, ok := request.UserFrom(r.Context())
	if !ok {
		return nil, newError(responses.ErrorGeneric, "Internal error")
	}

	user := buildUserResponse(loggedUser)
	response := newResponse()
	response.Users = &responses.Users{User: []responses.User{user}}
	return response, nil
}
