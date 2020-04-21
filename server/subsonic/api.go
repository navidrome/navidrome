package subsonic

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"net/http"
	"runtime"

	"github.com/deluan/navidrome/consts"
	"github.com/deluan/navidrome/engine"
	"github.com/deluan/navidrome/log"
	"github.com/deluan/navidrome/server/subsonic/responses"
	"github.com/deluan/navidrome/utils"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
)

const Version = "1.10.2"

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
	Streamer      engine.MediaStreamer
	Players       engine.Players

	mux http.Handler
}

func New(browser engine.Browser, cover engine.Cover, listGenerator engine.ListGenerator, users engine.Users,
	playlists engine.Playlists, ratings engine.Ratings, scrobbler engine.Scrobbler, search engine.Search,
	streamer engine.MediaStreamer, players engine.Players) *Router {

	r := &Router{Browser: browser, Cover: cover, ListGenerator: listGenerator, Playlists: playlists,
		Ratings: ratings, Scrobbler: scrobbler, Search: search, Users: users, Streamer: streamer, Players: players}
	r.mux = r.routes()
	return r
}

func (api *Router) Setup(path string) {}

func (api *Router) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	api.mux.ServeHTTP(w, r)
}

func (api *Router) routes() http.Handler {
	r := chi.NewRouter()

	r.Use(postFormToQueryParams)
	r.Use(checkRequiredParameters)
	r.Use(authenticate(api.Users))
	// TODO Validate version

	// Subsonic endpoints, grouped by controller
	r.Group(func(r chi.Router) {
		c := initSystemController(api)
		withPlayer := r.With(getPlayer(api.Players))
		H(withPlayer, "ping", c.Ping)
		H(withPlayer, "getLicense", c.GetLicense)
	})
	r.Group(func(r chi.Router) {
		c := initBrowsingController(api)
		withPlayer := r.With(getPlayer(api.Players))
		H(withPlayer, "getMusicFolders", c.GetMusicFolders)
		H(withPlayer, "getIndexes", c.GetIndexes)
		H(withPlayer, "getArtists", c.GetArtists)
		H(withPlayer, "getGenres", c.GetGenres)
		H(withPlayer, "getMusicDirectory", c.GetMusicDirectory)
		H(withPlayer, "getArtist", c.GetArtist)
		H(withPlayer, "getAlbum", c.GetAlbum)
		H(withPlayer, "getSong", c.GetSong)
		H(withPlayer, "getArtistInfo", c.GetArtistInfo)
		H(withPlayer, "getArtistInfo2", c.GetArtistInfo2)
	})
	r.Group(func(r chi.Router) {
		c := initAlbumListController(api)
		withPlayer := r.With(getPlayer(api.Players))
		H(withPlayer, "getAlbumList", c.GetAlbumList)
		H(withPlayer, "getAlbumList2", c.GetAlbumList2)
		H(withPlayer, "getStarred", c.GetStarred)
		H(withPlayer, "getStarred2", c.GetStarred2)
		H(withPlayer, "getNowPlaying", c.GetNowPlaying)
		H(withPlayer, "getRandomSongs", c.GetRandomSongs)
		H(withPlayer, "getSongsByGenre", c.GetSongsByGenre)
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
		withPlayer := r.With(getPlayer(api.Players))
		H(withPlayer, "getPlaylists", c.GetPlaylists)
		H(withPlayer, "getPlaylist", c.GetPlaylist)
		H(withPlayer, "createPlaylist", c.CreatePlaylist)
		H(withPlayer, "deletePlaylist", c.DeletePlaylist)
		H(withPlayer, "updatePlaylist", c.UpdatePlaylist)
	})
	r.Group(func(r chi.Router) {
		c := initSearchingController(api)
		withPlayer := r.With(getPlayer(api.Players))
		H(withPlayer, "search2", c.Search2)
		H(withPlayer, "search3", c.Search3)
	})
	r.Group(func(r chi.Router) {
		c := initUsersController(api)
		H(r, "getUser", c.GetUser)
	})
	r.Group(func(r chi.Router) {
		c := initMediaRetrievalController(api)
		// configure request throttling
		maxRequests := utils.MaxInt(2, runtime.NumCPU())
		withThrottle := r.With(middleware.ThrottleBacklog(maxRequests, consts.RequestThrottleBacklogLimit, consts.RequestThrottleBacklogTimeout))
		H(withThrottle, "getAvatar", c.GetAvatar)
		H(withThrottle, "getCoverArt", c.GetCoverArt)
	})
	r.Group(func(r chi.Router) {
		c := initStreamController(api)
		withPlayer := r.With(getPlayer(api.Players))
		H(withPlayer, "stream", c.Stream)
		H(withPlayer, "download", c.Download)
	})

	// Deprecated/Out of scope endpoints
	HGone(r, "getChatMessages")
	HGone(r, "addChatMessage")
	HGone(r, "getVideos")
	HGone(r, "getVideoInfo")
	HGone(r, "getCaptions")
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
	f := utils.ParamString(r, "f")
	var response []byte
	switch f {
	case "json":
		w.Header().Set("Content-Type", "application/json")
		wrapper := &responses.JsonWrapper{Subsonic: *payload}
		response, _ = json.Marshal(wrapper)
	case "jsonp":
		w.Header().Set("Content-Type", "application/javascript")
		callback := utils.ParamString(r, "callback")
		wrapper := &responses.JsonWrapper{Subsonic: *payload}
		data, _ := json.Marshal(wrapper)
		response = []byte(fmt.Sprintf("%s(%s)", callback, data))
	default:
		w.Header().Set("Content-Type", "application/xml")
		response, _ = xml.Marshal(payload)
	}
	if payload.Status == "ok" {
		if log.CurrentLevel() >= log.LevelTrace {
			log.Debug(r.Context(), "API: Successful response", "status", "OK", "body", string(response))
		} else {
			log.Debug(r.Context(), "API: Successful response", "status", "OK")
		}
	} else {
		log.Warn(r.Context(), "API: Failed response", "error", payload.Error.Code, "message", payload.Error.Message)
	}
	w.Write(response)
}
