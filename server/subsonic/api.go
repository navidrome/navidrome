package subsonic

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"net/http"
	"runtime"

	"github.com/deluan/navidrome/consts"
	"github.com/deluan/navidrome/core"
	"github.com/deluan/navidrome/log"
	"github.com/deluan/navidrome/model"
	"github.com/deluan/navidrome/server/subsonic/responses"
	"github.com/deluan/navidrome/utils"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
)

const Version = "1.13.0"

type handler = func(http.ResponseWriter, *http.Request) (*responses.Subsonic, error)

type Router struct {
	Artwork      core.Artwork
	Streamer     core.MediaStreamer
	Archiver     core.Archiver
	Players      core.Players
	ExternalInfo core.ExternalInfo
	DataStore    model.DataStore

	mux http.Handler
}

func New(artwork core.Artwork, streamer core.MediaStreamer, archiver core.Archiver, players core.Players,
	externalInfo core.ExternalInfo, ds model.DataStore) *Router {
	r := &Router{
		Artwork:      artwork,
		Streamer:     streamer,
		Archiver:     archiver,
		Players:      players,
		ExternalInfo: externalInfo,
		DataStore:    ds,
	}
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
	r.Use(authenticate(api.DataStore))
	// TODO Validate version

	// Subsonic endpoints, grouped by controller
	r.Group(func(r chi.Router) {
		c := initSystemController(api)
		withPlayer := r.With(getPlayer(api.Players))
		h(withPlayer, "ping", c.Ping)
		h(withPlayer, "getLicense", c.GetLicense)
	})
	r.Group(func(r chi.Router) {
		c := initBrowsingController(api)
		withPlayer := r.With(getPlayer(api.Players))
		h(withPlayer, "getMusicFolders", c.GetMusicFolders)
		h(withPlayer, "getIndexes", c.GetIndexes)
		h(withPlayer, "getArtists", c.GetArtists)
		h(withPlayer, "getGenres", c.GetGenres)
		h(withPlayer, "getMusicDirectory", c.GetMusicDirectory)
		h(withPlayer, "getArtist", c.GetArtist)
		h(withPlayer, "getAlbum", c.GetAlbum)
		h(withPlayer, "getSong", c.GetSong)
		h(withPlayer, "getArtistInfo", c.GetArtistInfo)
		h(withPlayer, "getArtistInfo2", c.GetArtistInfo2)
		h(withPlayer, "getTopSongs", c.GetTopSongs)
		h(withPlayer, "getSimilarSongs", c.GetSimilarSongs)
		h(withPlayer, "getSimilarSongs2", c.GetSimilarSongs2)
	})
	r.Group(func(r chi.Router) {
		c := initAlbumListController(api)
		withPlayer := r.With(getPlayer(api.Players))
		h(withPlayer, "getAlbumList", c.GetAlbumList)
		h(withPlayer, "getAlbumList2", c.GetAlbumList2)
		h(withPlayer, "getStarred", c.GetStarred)
		h(withPlayer, "getStarred2", c.GetStarred2)
		h(withPlayer, "getNowPlaying", c.GetNowPlaying)
		h(withPlayer, "getRandomSongs", c.GetRandomSongs)
		h(withPlayer, "getSongsByGenre", c.GetSongsByGenre)
	})
	r.Group(func(r chi.Router) {
		c := initMediaAnnotationController(api)
		h(r, "setRating", c.SetRating)
		h(r, "star", c.Star)
		h(r, "unstar", c.Unstar)
		h(r, "scrobble", c.Scrobble)
	})
	r.Group(func(r chi.Router) {
		c := initPlaylistsController(api)
		withPlayer := r.With(getPlayer(api.Players))
		h(withPlayer, "getPlaylists", c.GetPlaylists)
		h(withPlayer, "getPlaylist", c.GetPlaylist)
		h(withPlayer, "createPlaylist", c.CreatePlaylist)
		h(withPlayer, "deletePlaylist", c.DeletePlaylist)
		h(withPlayer, "updatePlaylist", c.UpdatePlaylist)
	})
	r.Group(func(r chi.Router) {
		c := initBookmarksController(api)
		withPlayer := r.With(getPlayer(api.Players))
		h(withPlayer, "getBookmarks", c.GetBookmarks)
		h(withPlayer, "createBookmark", c.CreateBookmark)
		h(withPlayer, "deleteBookmark", c.DeleteBookmark)
		h(withPlayer, "getPlayQueue", c.GetPlayQueue)
		h(withPlayer, "savePlayQueue", c.SavePlayQueue)
	})
	r.Group(func(r chi.Router) {
		c := initSearchingController(api)
		withPlayer := r.With(getPlayer(api.Players))
		h(withPlayer, "search2", c.Search2)
		h(withPlayer, "search3", c.Search3)
	})
	r.Group(func(r chi.Router) {
		c := initUsersController(api)
		h(r, "getUser", c.GetUser)
	})
	r.Group(func(r chi.Router) {
		c := initMediaRetrievalController(api)
		// configure request throttling
		maxRequests := utils.MaxInt(2, runtime.NumCPU())
		withThrottle := r.With(middleware.ThrottleBacklog(maxRequests, consts.RequestThrottleBacklogLimit, consts.RequestThrottleBacklogTimeout))
		h(withThrottle, "getAvatar", c.GetAvatar)
		h(withThrottle, "getCoverArt", c.GetCoverArt)
	})
	r.Group(func(r chi.Router) {
		c := initStreamController(api)
		withPlayer := r.With(getPlayer(api.Players))
		h(withPlayer, "stream", c.Stream)
		h(withPlayer, "download", c.Download)
	})

	// Deprecated/Out of scope endpoints
	h410(r, "getChatMessages")
	h410(r, "addChatMessage")
	h410(r, "getVideos")
	h410(r, "getVideoInfo")
	h410(r, "getCaptions")
	return r
}

// Add the Subsonic handler, with and without `.view` extension
// Ex: if path = `ping` it will create the routes `/ping` and `/ping.view`
func h(r chi.Router, path string, f handler) {
	handle := func(w http.ResponseWriter, r *http.Request) {
		res, err := f(w, r)
		if err != nil {
			// If it is not a Subsonic error, convert it to an ErrorGeneric
			if _, ok := err.(subError); !ok {
				err = newError(responses.ErrorGeneric, "Internal Error")
			}
			sendError(w, r, err)
			return
		}
		if res != nil {
			sendResponse(w, r, res)
		}
	}
	r.HandleFunc("/"+path, handle)
	r.HandleFunc("/"+path+".view", handle)
}

// Add a handler that returns 410 - Gone. Used to signal that an endpoint will not be implemented
func h410(r chi.Router, path string) {
	handle := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(410)
		_, _ = w.Write([]byte("This endpoint will not be implemented"))
	}
	r.HandleFunc("/"+path, handle)
	r.HandleFunc("/"+path+".view", handle)
}

func sendError(w http.ResponseWriter, r *http.Request, err error) {
	response := newResponse()
	code := responses.ErrorGeneric
	if e, ok := err.(subError); ok {
		code = e.code
	}
	response.Status = "fail"
	response.Error = &responses.Error{Code: code, Message: err.Error()}

	sendResponse(w, r, response)
}

func sendResponse(w http.ResponseWriter, r *http.Request, payload *responses.Subsonic) {
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
	if _, err := w.Write(response); err != nil {
		log.Error(r, "Error sending response to client", "payload", string(response), err)
	}
}
