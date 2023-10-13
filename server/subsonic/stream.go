package subsonic

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/core"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
	"github.com/navidrome/navidrome/server/subsonic/responses"
	"github.com/navidrome/navidrome/utils/req"
)

type BeetsItem struct {
	Results []struct {
		Artpath              string  `json:"artpath"`
		Year                 int     `json:"year"`
		Albumtype            string  `json:"albumtype"`
		R128TrackGain        any     `json:"r128_track_gain"`
		Day                  int     `json:"day"`
		Disc                 int     `json:"disc"`
		Albumartist          string  `json:"albumartist"`
		MbReleasegroupid     string  `json:"mb_releasegroupid"`
		InitialKey           any     `json:"initial_key"`
		AcoustidID           string  `json:"acoustid_id"`
		MbAlbumid            string  `json:"mb_albumid"`
		Month                int     `json:"month"`
		Track                int     `json:"track"`
		Style                string  `json:"style"`
		RgAlbumGain          any     `json:"rg_album_gain"`
		Album                string  `json:"album"`
		DiscogsArtistid      int     `json:"discogs_artistid"`
		Tracktotal           int     `json:"tracktotal"`
		MbAlbumartistid      string  `json:"mb_albumartistid"`
		Country              string  `json:"country"`
		Channels             int     `json:"channels"`
		Arranger             string  `json:"arranger"`
		Comp                 int     `json:"comp"`
		Bitrate              int     `json:"bitrate"`
		Length               float64 `json:"length"`
		Lyricist             string  `json:"lyricist"`
		RgAlbumPeak          any     `json:"rg_album_peak"`
		MbArtistid           string  `json:"mb_artistid"`
		Trackdisambig        string  `json:"trackdisambig"`
		OriginalMonth        int     `json:"original_month"`
		AcoustidFingerprint  string  `json:"acoustid_fingerprint"`
		OriginalDay          int     `json:"original_day"`
		ComposerSort         string  `json:"composer_sort"`
		AlbumartistCredit    string  `json:"albumartist_credit"`
		MbWorkid             string  `json:"mb_workid"`
		MbReleasetrackid     string  `json:"mb_releasetrackid"`
		Mtime                float64 `json:"mtime"`
		Work                 string  `json:"work"`
		AlbumartistSort      string  `json:"albumartist_sort"`
		DiscogsLabelid       int     `json:"discogs_labelid"`
		Bpm                  int     `json:"bpm"`
		Language             string  `json:"language"`
		DataSource           string  `json:"data_source"`
		ArtistCredit         string  `json:"artist_credit"`
		Format               string  `json:"format"`
		Composer             string  `json:"composer"`
		Disctotal            int     `json:"disctotal"`
		Title                string  `json:"title"`
		Grouping             string  `json:"grouping"`
		Added                float64 `json:"added"`
		Media                string  `json:"media"`
		Artist               string  `json:"artist"`
		Albumdisambig        string  `json:"albumdisambig"`
		Isrc                 string  `json:"isrc"`
		DiscogsAlbumid       int     `json:"discogs_albumid"`
		Disctitle            string  `json:"disctitle"`
		Lyrics               string  `json:"lyrics"`
		Albumstatus          string  `json:"albumstatus"`
		Albumtypes           string  `json:"albumtypes"`
		Releasegroupdisambig string  `json:"releasegroupdisambig"`
		Comments             string  `json:"comments"`
		Encoder              string  `json:"encoder"`
		Catalognum           string  `json:"catalognum"`
		R128AlbumGain        any     `json:"r128_album_gain"`
		Label                string  `json:"label"`
		Bitdepth             int     `json:"bitdepth"`
		RgTrackGain          any     `json:"rg_track_gain"`
		ID                   int     `json:"id"`
		Script               string  `json:"script"`
		ArtistSort           string  `json:"artist_sort"`
		TrackAlt             string  `json:"track_alt"`
		Genre                string  `json:"genre"`
		OriginalYear         int     `json:"original_year"`
		WorkDisambig         string  `json:"work_disambig"`
		AlbumID              int     `json:"album_id"`
		MbTrackid            string  `json:"mb_trackid"`
		Asin                 string  `json:"asin"`
		RgTrackPeak          any     `json:"rg_track_peak"`
		Samplerate           int     `json:"samplerate"`
		Size                 int     `json:"size"`
	} `json:"results"`
}

func (api *Router) serveStream(ctx context.Context, w http.ResponseWriter, r *http.Request, stream *core.Stream, id string) {
	if stream.Seekable() {
		http.ServeContent(w, r, stream.Name(), stream.ModTime(), stream)
	} else {
		// If the stream doesn't provide a size (i.e. is not seekable), we can't support ranges/content-length
		w.Header().Set("Accept-Ranges", "none")
		w.Header().Set("Content-Type", stream.ContentType())

		estimateContentLength := req.Params(r).BoolOr("estimateContentLength", false)

		// if Client requests the estimated content-length, send it
		if estimateContentLength {
			length := strconv.Itoa(stream.EstimatedContentLength())
			log.Trace(ctx, "Estimated content-length", "contentLength", length)
			w.Header().Set("Content-Length", length)
		}

		if r.Method == http.MethodHead {
			go func() { _, _ = io.Copy(io.Discard, stream) }()
		} else {
			c, err := io.Copy(w, stream)
			if log.CurrentLevel() >= log.LevelDebug {
				if err != nil {
					log.Error(ctx, "Error sending transcoded file", "id", id, err)
				} else {
					log.Trace(ctx, "Success sending transcode file", "id", id, "size", c)
				}
			}
		}
	}
}

func (api *Router) Stream(w http.ResponseWriter, r *http.Request) (*responses.Subsonic, error) {
	ctx := r.Context()
	p := req.Params(r)
	id, err := p.String("id")
	if err != nil {
		return nil, err
	}
	maxBitRate := p.IntOr("maxBitRate", 0)
	format, _ := p.String("format")
	timeOffset := p.IntOr("timeOffset", 0)

	stream, err := api.streamer.NewStream(ctx, id, format, maxBitRate, timeOffset)
	if err != nil {
		return nil, err
	}

	// Make sure the stream will be closed at the end, to avoid leakage
	defer func() {
		if err := stream.Close(); err != nil && log.CurrentLevel() >= log.LevelDebug {
			log.Error("Error closing stream", "id", id, "file", stream.Name(), err)
		}
	}()

	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("X-Content-Duration", strconv.FormatFloat(float64(stream.Duration()), 'G', -1, 32))

	api.serveStream(ctx, w, r, stream, id)

	return nil, nil
}

func (api *Router) Delete(w http.ResponseWriter, r *http.Request) (*responses.Subsonic, error) {
	// todo: tests, use proper url parsing lib
	println("hello from go. Deleting...")
	id, err := requiredParamString(r, "id")
	ctx := r.Context()
	ids := strings.Split(id, ",")
	for _, id := range ids {
		println(id)
		mf, err := api.ds.MediaFile(ctx).Get(id)
		if err != nil {
			log.Error(err)
		}
		println(mf.Artist)
		println(mf.Title)
		// todo set this base from env variable
		//baseUrl := "http://127.0.0.1:8337"
		baseUrl := "http://host.docker.internal:8337"
		queryEndPoint := "/item/query/"
		queryStr := fmt.Sprintf("artist:%s/title:%s", mf.Artist, mf.Title)
		url := baseUrl + queryEndPoint + queryStr
		fmt.Printf("query url: %s\n", url)
		resp, err := http.Get(url) // nolint
		if err != nil {
			log.Error(err)
			return nil, err
		}
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Error(err)
			return nil, err
		}
		sb := string(body)
		var beetsItem BeetsItem
		err = json.Unmarshal([]byte(sb), &beetsItem)
		if err != nil {
			log.Error(err)
			return nil, err
		}
		length := len(beetsItem.Results)
		if length != 1 {
			log.Error("following query string matched: ", length, "tracks", queryStr)
			return nil, err
		}
		item := beetsItem.Results[0]
		log.Info("deleting: ", item.Artist, " : ", item.Title)
		// Create request
		del_url := url + "?delete"
		req, err := http.NewRequest(http.MethodDelete, del_url, nil)
		log.Info("delete request: ", req)
		if err != nil {
			log.Error(err)
			return nil, err
		}
		// Fetch Request
		client := &http.Client{}
		del_resp, del_err := client.Do(req)
		if del_err != nil {
			log.Error(err)
			return nil, del_err
		}
		defer del_resp.Body.Close()

		// read the response body
		del_body, err := io.ReadAll(del_resp.Body)
		if err != nil {
			log.Error(err)
			return nil, del_err
		}

		// print the response body
		fmt.Println(string(del_body))
		log.Info("del body: ", del_body)
	}
	return nil, err
}

func (api *Router) Download(w http.ResponseWriter, r *http.Request) (*responses.Subsonic, error) {
	ctx := r.Context()
	username, _ := request.UsernameFrom(ctx)
	p := req.Params(r)
	id, err := p.String("id")
	if err != nil {
		return nil, err
	}

	if !conf.Server.EnableDownloads {
		log.Warn(ctx, "Downloads are disabled", "user", username, "id", id)
		return nil, newError(responses.ErrorAuthorizationFail, "downloads are disabled")
	}

	entity, err := model.GetEntityByID(ctx, api.ds, id)
	if err != nil {
		return nil, err
	}

	maxBitRate := p.IntOr("bitrate", 0)
	format, _ := p.String("format")

	if format == "" {
		if conf.Server.AutoTranscodeDownload {
			// if we are not provided a format, see if we have requested transcoding for this client
			// This must be enabled via a config option. For the UI, we are always given an option.
			// This will impact other clients which do not use the UI
			transcoding, ok := request.TranscodingFrom(ctx)

			if !ok {
				format = "raw"
			} else {
				format = transcoding.TargetFormat
				maxBitRate = transcoding.DefaultBitRate
			}
		} else {
			format = "raw"
		}
	}

	setHeaders := func(name string) {
		name = strings.ReplaceAll(name, ",", "_")
		disposition := fmt.Sprintf("attachment; filename=\"%s.zip\"", name)
		w.Header().Set("Content-Disposition", disposition)
		w.Header().Set("Content-Type", "application/zip")
	}

	switch v := entity.(type) {
	case *model.MediaFile:
		stream, err := api.streamer.NewStream(ctx, id, format, maxBitRate, 0)
		if err != nil {
			return nil, err
		}

		// Make sure the stream will be closed at the end, to avoid leakage
		defer func() {
			if err := stream.Close(); err != nil && log.CurrentLevel() >= log.LevelDebug {
				log.Error("Error closing stream", "id", id, "file", stream.Name(), err)
			}
		}()

		disposition := fmt.Sprintf("attachment; filename=\"%s\"", stream.Name())
		w.Header().Set("Content-Disposition", disposition)

		api.serveStream(ctx, w, r, stream, id)
		return nil, nil
	case *model.Album:
		setHeaders(v.Name)
		err = api.archiver.ZipAlbum(ctx, id, format, maxBitRate, w)
	case *model.Artist:
		setHeaders(v.Name)
		err = api.archiver.ZipArtist(ctx, id, format, maxBitRate, w)
	case *model.Playlist:
		setHeaders(v.Name)
		err = api.archiver.ZipPlaylist(ctx, id, format, maxBitRate, w)
	default:
		err = model.ErrNotFound
	}

	return nil, err
}
