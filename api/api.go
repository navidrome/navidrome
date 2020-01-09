package api

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"net/http"

	"github.com/cloudsonic/sonic-server/api/responses"
	"github.com/cloudsonic/sonic-server/conf"
	"github.com/go-chi/chi"
)

const ApiVersion = "1.8.0"

type SubsonicHandler = func(http.ResponseWriter, *http.Request) (*responses.Subsonic, error)

func Router() http.Handler {
	r := chi.NewRouter()

	r.Use(checkRequiredParameters)

	// Add validation middleware if not disabled
	if !conf.Sonic.DisableAuthentication {
		r.Use(authenticate)
		// TODO Validate version
	}

	r.Group(func(r chi.Router) {
		c := initSystemController()
		addEndpoint(r, "ping", c.Ping)
		addEndpoint(r, "getLicense", c.GetLicense)
	})
	r.Group(func(r chi.Router) {
		c := initBrowsingController()
		addEndpoint(r, "getMusicFolders", c.GetMusicFolders)
		addEndpoint(r, "getMusicFolders", c.GetMusicFolders)
		addEndpoint(r, "getIndexes", c.GetIndexes)
		addEndpoint(r, "getArtists", c.GetArtists)
		reqParams := r.With(requiredParams("id"))
		addEndpoint(reqParams, "getMusicDirectory", c.GetMusicDirectory)
		addEndpoint(reqParams, "getArtist", c.GetArtist)
		addEndpoint(reqParams, "getAlbum", c.GetAlbum)
		addEndpoint(reqParams, "getSong", c.GetSong)
	})
	r.Group(func(r chi.Router) {
		c := initAlbumListController()
		addEndpoint(r, "getAlbumList", c.GetAlbumList)
		addEndpoint(r, "getAlbumList2", c.GetAlbumList2)
		addEndpoint(r, "getStarred", c.GetStarred)
		addEndpoint(r, "getStarred2", c.GetStarred2)
		addEndpoint(r, "getNowPlaying", c.GetNowPlaying)
		addEndpoint(r, "getRandomSongs", c.GetRandomSongs)
	})
	r.Group(func(r chi.Router) {
		c := initMediaAnnotationController()
		addEndpoint(r, "setRating", c.SetRating)
		addEndpoint(r, "star", c.Star)
		addEndpoint(r, "unstar", c.Unstar)
		addEndpoint(r, "scrobble", c.Scrobble)
	})
	r.Group(func(r chi.Router) {
		c := initPlaylistsController()
		addEndpoint(r, "getPlaylists", c.GetPlaylists)
		addEndpoint(r, "getPlaylist", c.GetPlaylist)
		addEndpoint(r, "createPlaylist", c.CreatePlaylist)
		addEndpoint(r, "deletePlaylist", c.DeletePlaylist)
		addEndpoint(r, "updatePlaylist", c.UpdatePlaylist)
	})
	r.Group(func(r chi.Router) {
		c := initSearchingController()
		addEndpoint(r, "search2", c.Search2)
		addEndpoint(r, "search3", c.Search3)
	})
	r.Group(func(r chi.Router) {
		c := initUsersController()
		addEndpoint(r, "getUser", c.GetUser)
	})
	r.Group(func(r chi.Router) {
		c := initMediaRetrievalController()
		addEndpoint(r, "getAvatar", c.GetAvatar)
		addEndpoint(r, "getCoverArt", c.GetCoverArt)
	})
	r.Group(func(r chi.Router) {
		c := initStreamController()
		addEndpoint(r, "stream", c.Stream)
		addEndpoint(r, "download", c.Download)
	})
	return r
}

func addEndpoint(r chi.Router, path string, f SubsonicHandler) {
	handle := func(w http.ResponseWriter, r *http.Request) {
		res, err := f(w, r)
		if err != nil {
			SendError(w, r, err)
			return
		}
		if res != nil {
			SendResponse(w, r, res)
		}
	}
	r.HandleFunc("/"+path, handle)
	r.HandleFunc("/"+path+".view", handle)
}

func SendError(w http.ResponseWriter, r *http.Request, err error) {
	response := &responses.Subsonic{Version: ApiVersion, Status: "fail"}
	code := responses.ErrorGeneric
	if e, ok := err.(SubsonicError); ok {
		code = e.code
	}
	response.Error = &responses.Error{Code: code, Message: err.Error()}

	SendResponse(w, r, response)
}

func SendResponse(w http.ResponseWriter, r *http.Request, payload *responses.Subsonic) {
	f := ParamString(r, "f")
	var response []byte
	switch f {
	case "json":
		w.Header().Set("Content-Type", "application/json")
		wrapper := &responses.JsonWrapper{Subsonic: *payload}
		response, _ = json.Marshal(wrapper)
	case "jsonp":
		w.Header().Set("Content-Type", "application/javascript")
		callback := ParamString(r, "callback")
		wrapper := &responses.JsonWrapper{Subsonic: *payload}
		data, _ := json.Marshal(wrapper)
		response = []byte(fmt.Sprintf("%s(%s)", callback, data))
	default:
		w.Header().Set("Content-Type", "application/xml")
		response, _ = xml.Marshal(payload)
	}
	w.Write(response)
}
