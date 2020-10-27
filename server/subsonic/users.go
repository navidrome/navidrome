package subsonic

import (
	"net/http"

	"github.com/deluan/navidrome/server/subsonic/responses"
)

type UsersController struct{}

func NewUsersController() *UsersController {
	return &UsersController{}
}

// TODO This is a placeholder. The real one has to read this info from a config file or the database
func (c *UsersController) GetUser(w http.ResponseWriter, r *http.Request) (*responses.Subsonic, error) {
	user, err := requiredParamString(r, "username")
	if err != nil {
		return nil, err
	}
	response := newResponse()
	response.User = &responses.User{}
	response.User.Username = user
	response.User.StreamRole = true
	response.User.DownloadRole = true
	response.User.ScrobblingEnabled = true
	return response, nil
}
