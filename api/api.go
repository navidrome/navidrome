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

	// Add validation middleware if not disabled
	if !conf.Sonic.DisableValidation {
		r.Use(checkRequiredParameters)
		r.Use(authenticate)
		// TODO Validate version
	}

	r.Group(func(r chi.Router) {
		c := initSystemController()
		r.HandleFunc("/ping.view", addMethod(c.Ping))
		r.HandleFunc("/getLicense.view", addMethod(c.GetLicense))
	})
	r.Group(func(r chi.Router) {
		c := initBrowsingController()
		r.HandleFunc("/getMusicFolders.view", addMethod(c.GetMusicFolders))
		r.HandleFunc("/getIndexes.view", addMethod(c.GetIndexes))
		r.HandleFunc("/getArtists.view", addMethod(c.GetArtists))
		r.With(requiredParams("id")).HandleFunc("/getMusicDirectory.view", addMethod(c.GetMusicDirectory))
		r.With(requiredParams("id")).HandleFunc("/getArtist.view", addMethod(c.GetArtist))
		r.With(requiredParams("id")).HandleFunc("/getAlbum.view", addMethod(c.GetAlbum))
		r.With(requiredParams("id")).HandleFunc("/getSong.view", addMethod(c.GetSong))
	})
	r.Group(func(r chi.Router) {
		c := initAlbumListController()
		r.HandleFunc("/getAlbumList.view", addMethod(c.GetAlbumList))
		r.HandleFunc("/getAlbumList2.view", addMethod(c.GetAlbumList2))
		r.HandleFunc("/getStarred.view", addMethod(c.GetStarred))
		r.HandleFunc("/getStarred2.view", addMethod(c.GetStarred2))
		r.HandleFunc("/getNowPlaying.view", addMethod(c.GetNowPlaying))
		r.HandleFunc("/getRandomSongs.view", addMethod(c.GetRandomSongs))
	})
	r.Group(func(r chi.Router) {
		c := initMediaAnnotationController()
		r.HandleFunc("/setRating.view", addMethod(c.SetRating))
		r.HandleFunc("/star.view", addMethod(c.Star))
		r.HandleFunc("/unstar.view", addMethod(c.Unstar))
		r.HandleFunc("/scrobble.view", addMethod(c.Scrobble))
	})
	r.Group(func(r chi.Router) {
		c := initPlaylistsController()
		r.HandleFunc("/getPlaylists.view", addMethod(c.GetPlaylists))
		r.HandleFunc("/getPlaylist.view", addMethod(c.GetPlaylist))
		r.HandleFunc("/createPlaylist.view", addMethod(c.CreatePlaylist))
		r.HandleFunc("/deletePlaylist.view", addMethod(c.DeletePlaylist))
		r.HandleFunc("/updatePlaylist.view", addMethod(c.UpdatePlaylist))
	})
	r.Group(func(r chi.Router) {
		c := initSearchingController()
		r.HandleFunc("/search2.view", addMethod(c.Search2))
		r.HandleFunc("/search3.view", addMethod(c.Search3))
	})
	r.Group(func(r chi.Router) {
		c := initUsersController()
		r.HandleFunc("/getUser.view", addMethod(c.GetUser))
	})
	r.Group(func(r chi.Router) {
		c := initMediaRetrievalController()
		r.HandleFunc("/getAvatar.view", addMethod(c.GetAvatar))
		r.HandleFunc("/getCoverArt.view", addMethod(c.GetCoverArt))
	})
	r.Group(func(r chi.Router) {
		c := initStreamController()
		r.HandleFunc("/stream.view", addMethod(c.Stream))
		r.HandleFunc("/download.view", addMethod(c.Download))
	})
	return r
}

func addMethod(method SubsonicHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		res, err := method(w, r)
		if err != nil {
			SendError(w, r, err)
			return
		}
		if res != nil {
			SendResponse(w, r, res)
		}
	}
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
		w.Header().Set("Content-Type", "application/json")
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
