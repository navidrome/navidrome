package subsonic

import (
	"net/http"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/model/request"
	"github.com/navidrome/navidrome/server/subsonic/responses"
)

// TODO This is a placeholder. The real one has to read this info from a config file or the database
func (api *Router) GetUser(r *http.Request) (*responses.Subsonic, error) {
	loggedUser, ok := request.UserFrom(r.Context())
	if !ok {
		return nil, newError(responses.ErrorGeneric, "Internal error")
	}
	response := newResponse()
	response.User = &responses.User{}
	response.User.Username = loggedUser.UserName
	response.User.AdminRole = loggedUser.IsAdmin
	response.User.Email = loggedUser.Email
	response.User.SyncPlayqueue = loggedUser.SyncPlayqueue
	response.User.StreamRole = true
	response.User.ScrobblingEnabled = true
	response.User.DownloadRole = conf.Server.EnableDownloads
	response.User.ShareRole = conf.Server.EnableSharing
	response.User.JukeboxRole = conf.Server.Jukebox.Enabled
	return response, nil
}

func (api *Router) GetUsers(r *http.Request) (*responses.Subsonic, error) {
	loggedUser, ok := request.UserFrom(r.Context())
	if !ok {
		return nil, newError(responses.ErrorGeneric, "Internal error")
	}
	user := responses.User{}
	user.Username = loggedUser.Name
	user.AdminRole = loggedUser.IsAdmin
	user.Email = loggedUser.Email
	user.StreamRole = true
	user.ScrobblingEnabled = true
	user.DownloadRole = conf.Server.EnableDownloads
	user.ShareRole = conf.Server.EnableSharing
	user.JukeboxRole = conf.Server.Jukebox.Enabled
	user.SyncPlayqueue = loggedUser.SyncPlayqueue
	response := newResponse()
	response.Users = &responses.Users{User: []responses.User{user}}
	return response, nil
}
