package subsonic

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"net/http"

	"github.com/deluan/navidrome/engine"
	"github.com/deluan/navidrome/server/subsonic/responses"
	"github.com/go-chi/chi"
)

const Version = "1.8.0"

type Handler = func(http.ResponseWriter, *http.Request) (*responses.Subsonic, error)

type Router struct {
	Browser       engine.Browser
	Cover         engine.Cover
	ListGenerator engine.ListGenerator
	Playlists     engine.Playlists
	Ratings       engine.Ratings
	Scrobbler     engine.Scrobbler
	Search        engine.Search
	Users         engine.Users

	mux http.Handler
}

func New(browser engine.Browser, cover engine.Cover, listGenerator engine.ListGenerator, users engine.Users,
	playlists engine.Playlists, ratings engine.Ratings, scrobbler engine.Scrobbler, search engine.Search) *Router {

	r := &Router{Browser: browser, Cover: cover, ListGenerator: listGenerator, Playlists: playlists,
		Ratings: ratings, Scrobbler: scrobbler, Search: search, Users: users}
	r.mux = r.routes()
	return r
}

func (api *Router) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	api.mux.ServeHTTP(w, r)
}

func (api *Router) routes() http.Handler {
	r := chi.NewRouter()

	r.Use(postFormToQueryParams)
	r.Use(checkRequiredParameters)

	// Add validation middleware
	r.Use(authenticate(api.Users))
	// TODO Validate version

	// Subsonic endpoints, grouped by controller
	r.Group(func(r chi.Router) {
		c := initSystemController(api)
		H(r, "ping", c.Ping)
		H(r, "getLicense", c.GetLicense)
	})
	r.Group(func(r chi.Router) {
		c := initBrowsingController(api)
		H(r, "getMusicFolders", c.GetMusicFolders)
		H(r, "getMusicFolders", c.GetMusicFolders)
		H(r, "getIndexes", c.GetIndexes)
		H(r, "getArtists", c.GetArtists)
		H(r, "getGenres", c.GetGenres)
		reqParams := r.With(requiredParams("id"))
		H(reqParams, "getMusicDirectory", c.GetMusicDirectory)
		H(reqParams, "getArtist", c.GetArtist)
		H(reqParams, "getAlbum", c.GetAlbum)
		H(reqParams, "getSong", c.GetSong)
	})
	r.Group(func(r chi.Router) {
		c := initAlbumListController(api)
		H(r, "getAlbumList", c.GetAlbumList)
		H(r, "getAlbumList2", c.GetAlbumList2)
		H(r, "getStarred", c.GetStarred)
		H(r, "getStarred2", c.GetStarred2)
		H(r, "getNowPlaying", c.GetNowPlaying)
		H(r, "getRandomSongs", c.GetRandomSongs)
	})
	r.Group(func(r chi.Router) {
		c := initMediaAnnotationController(api)
		H(r, "setRating", c.SetRating)
		H(r, "star", c.Star)
		H(r, "unstar", c.Unstar)
		H(r, "scrobble", c.Scrobble)
	})
	r.Group(func(r chi.Router) {
		c := initPlaylistsController(api)
		H(r, "getPlaylists", c.GetPlaylists)
		H(r, "getPlaylist", c.GetPlaylist)
		H(r, "createPlaylist", c.CreatePlaylist)
		H(r, "deletePlaylist", c.DeletePlaylist)
		H(r, "updatePlaylist", c.UpdatePlaylist)
	})
	r.Group(func(r chi.Router) {
		c := initSearchingController(api)
		H(r, "search2", c.Search2)
		H(r, "search3", c.Search3)
	})
	r.Group(func(r chi.Router) {
		c := initUsersController(api)
		H(r, "getUser", c.GetUser)
	})
	r.Group(func(r chi.Router) {
		c := initMediaRetrievalController(api)
		H(r, "getAvatar", c.GetAvatar)
		H(r, "getCoverArt", c.GetCoverArt)
	})
	r.Group(func(r chi.Router) {
		c := initStreamController(api)
		H(r, "stream", c.Stream)
		H(r, "download", c.Download)
	})

	// Deprecated/Out of scope endpoints
	HGone(r, "getChatMessages")
	HGone(r, "addChatMessage")
	return r
}

// Add the Subsonic handler, with and without `.view` extension
// Ex: if path = `ping` it will create the routes `/ping` and `/ping.view`
func H(r chi.Router, path string, f Handler) {
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

// Add a handler that returns 410 - Gone. Used to signal that an endpoint will not be implemented
func HGone(r chi.Router, path string) {
	handle := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(410)
		w.Write([]byte("This endpoint will not be implemented"))
	}
	r.HandleFunc("/"+path, handle)
	r.HandleFunc("/"+path+".view", handle)
}

func SendError(w http.ResponseWriter, r *http.Request, err error) {
	response := NewResponse()
	code := responses.ErrorGeneric
	if e, ok := err.(SubsonicError); ok {
		code = e.code
	}
	response.Status = "fail"
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
