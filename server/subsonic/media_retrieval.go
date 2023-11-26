package subsonic

import (
	"context"
	"errors"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/resources"
	"github.com/navidrome/navidrome/server/subsonic/filter"
	"github.com/navidrome/navidrome/server/subsonic/responses"
	"github.com/navidrome/navidrome/utils"
	"github.com/navidrome/navidrome/utils/gravatar"
)

func (api *Router) GetAvatar(w http.ResponseWriter, r *http.Request) (*responses.Subsonic, error) {
	if !conf.Server.EnableGravatar {
		return api.getPlaceHolderAvatar(w, r)
	}
	username, err := requiredParamString(r, "username")
	if err != nil {
		return nil, err
	}
	ctx := r.Context()
	u, err := api.ds.User(ctx).FindByUsername(username)
	if err != nil {
		return nil, err
	}
	if u.Email == "" {
		log.Warn(ctx, "User needs an email for gravatar to work", "username", username)
		return api.getPlaceHolderAvatar(w, r)
	}
	http.Redirect(w, r, gravatar.Url(u.Email, 0), http.StatusFound)
	return nil, nil
}

func (api *Router) getPlaceHolderAvatar(w http.ResponseWriter, r *http.Request) (*responses.Subsonic, error) {
	f, err := resources.FS().Open(consts.PlaceholderAvatar)
	if err != nil {
		log.Error(r, "Image not found", err)
		return nil, newError(responses.ErrorDataNotFound, "Avatar image not found")
	}
	defer f.Close()
	_, _ = io.Copy(w, f)

	return nil, nil
}

func (api *Router) GetCoverArt(w http.ResponseWriter, r *http.Request) (*responses.Subsonic, error) {
	// If context is already canceled, discard request without further processing
	if r.Context().Err() != nil {
		return nil, nil //nolint:nilerr
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	id := utils.ParamString(r, "id")
	size := utils.ParamInt(r, "size", 0)

	imgReader, lastUpdate, err := api.artwork.GetOrPlaceholder(ctx, id, size)
	w.Header().Set("cache-control", "public, max-age=315360000")
	w.Header().Set("last-modified", lastUpdate.Format(time.RFC1123))

	switch {
	case errors.Is(err, context.Canceled):
		return nil, nil
	case errors.Is(err, model.ErrNotFound):
		log.Warn(r, "Couldn't find coverArt", "id", id, err)
		return nil, newError(responses.ErrorDataNotFound, "Artwork not found")
	case err != nil:
		log.Error(r, "Error retrieving coverArt", "id", id, err)
		return nil, err
	}

	defer imgReader.Close()
	cnt, err := io.Copy(w, imgReader)
	if err != nil {
		log.Warn(ctx, "Error sending image", "count", cnt, err)
	}

	return nil, err
}

const timeRegexString = `(\[(([0-9]{1,2}):)?([0-9]{1,2}):([0-9]{1,2})(\.([0-9]{1,3}))?\])`

var (
	timeRegex  = regexp.MustCompile(timeRegexString)
	lineRegex  = regexp.MustCompile(timeRegexString + "([^\n]+)")
	lrcIdRegex = regexp.MustCompile(`\[(ar|ti|offset):([^\]]+)\]`)
)

func isSynced(rawLyrics string) bool {
	// Eg: [04:02:50.85]
	// [02:50.85]
	// [02:50]
	return timeRegex.MatchString(rawLyrics)
}

func (api *Router) GetLyrics(r *http.Request) (*responses.Subsonic, error) {
	artist := utils.ParamString(r, "artist")
	title := utils.ParamString(r, "title")
	response := newResponse()
	lyrics := responses.Lyrics{}
	response.Lyrics = &lyrics
	mediaFiles, err := api.ds.MediaFile(r.Context()).GetAll(filter.SongsWithLyrics(artist, title))

	if err != nil {
		return nil, err
	}

	if len(mediaFiles) == 0 {
		return response, nil
	}

	lyrics.Artist = artist
	lyrics.Title = title

	if isSynced(mediaFiles[0].Lyrics) {
		lyrics.Value = timeRegex.ReplaceAllString(mediaFiles[0].Lyrics, "")
	} else {
		lyrics.Value = mediaFiles[0].Lyrics
	}

	return response, nil
}

func (api *Router) GetLyricsBySongId(r *http.Request) (*responses.Subsonic, error) {
	id := utils.ParamString(r, "id")
	mediaFile, err := api.ds.MediaFile(r.Context()).Get(id)

	if err != nil {
		return nil, err
	}

	response := newResponse()
	allLyrics := responses.LyricsList{}
	response.LyricsList = &allLyrics

	if mediaFile.Lyrics != "" {
		lyricsResult := strings.Split(mediaFile.Lyrics, consts.Zwsp)

		for i := 0; i < len(lyricsResult); i += 2 {
			encoding := lyricsResult[i]
			textLines := strings.Split(lyricsResult[i+1], "\n")

			lines := []responses.Line{}
			synced := true

			artist := ""
			title := ""
			var offset *int64 = nil

			for _, line := range textLines {
				line := strings.TrimSpace(line)

				if line == "" {
					continue
				}

				var text string
				var time *int64 = nil

				if synced {
					idTag := lrcIdRegex.FindStringSubmatch(line)
					if idTag != nil {
						switch idTag[1] {
						case "ar":
							artist = idTag[2]
						case "offset":
							{
								off, err := strconv.ParseInt(idTag[2], 10, 64)
								if err != nil {
									return nil, err
								}

								offset = &off
							}
						case "ti":
							title = idTag[2]
						}

						continue
					}

					syncedMatch := lineRegex.FindStringSubmatch(line)
					if syncedMatch == nil {
						synced = false
						text = line
					} else {
						var hours int64
						if syncedMatch[3] != "" {
							hours, err = strconv.ParseInt(syncedMatch[3], 10, 64)
							if err != nil {
								return nil, err
							}
						}

						min, err := strconv.ParseInt(syncedMatch[4], 10, 64)
						if err != nil {
							return nil, err
						}

						sec, err := strconv.ParseInt(syncedMatch[5], 10, 64)
						if err != nil {
							return nil, err
						}

						millis, err := strconv.ParseInt(syncedMatch[7], 10, 64)
						if err != nil {
							return nil, err
						}

						if len(syncedMatch[7]) == 2 {
							millis *= 10
						}

						timeInMillis := (((((hours * 60) + min) * 60) + sec) * 1000) + millis
						time = &timeInMillis
						text = syncedMatch[8]
					}
				} else {
					text = line
				}

				lines = append(lines, responses.Line{
					Start: time,
					Value: text,
				})
			}

			response.LyricsList.StructuredLyrics = append(response.LyricsList.StructuredLyrics, responses.StructuredLyrics{
				DisplayArtist: artist,
				DisplayTitle:  title,
				Lang:          encoding,
				Line:          lines,
				Offset:        offset,
				Synced:        synced,
			})
		}
	}

	return response, nil
}
