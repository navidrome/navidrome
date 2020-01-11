package api

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/cloudsonic/sonic-server/api/responses"
	"github.com/cloudsonic/sonic-server/engine"
	"github.com/cloudsonic/sonic-server/utils"
)

func NewEmpty() *responses.Subsonic {
	return &responses.Subsonic{Status: "ok", Version: Version}
}

func RequiredParamString(r *http.Request, param string, msg string) (string, error) {
	p := ParamString(r, param)
	if p == "" {
		return "", NewError(responses.ErrorMissingParameter, msg)
	}
	return p, nil
}

func RequiredParamStrings(r *http.Request, param string, msg string) ([]string, error) {
	ps := ParamStrings(r, param)
	if len(ps) == 0 {
		return nil, NewError(responses.ErrorMissingParameter, msg)
	}
	return ps, nil
}

func ParamString(r *http.Request, param string) string {
	return r.URL.Query().Get(param)
}

func ParamStrings(r *http.Request, param string) []string {
	return r.URL.Query()[param]
}

func ParamTimes(r *http.Request, param string) []time.Time {
	pStr := ParamStrings(r, param)
	times := make([]time.Time, len(pStr))
	for i, t := range pStr {
		ti, err := strconv.ParseInt(t, 10, 64)
		if err == nil {
			times[i] = utils.ToTime(ti)
		}
	}
	return times
}

func ParamTime(r *http.Request, param string, def time.Time) time.Time {
	v := ParamString(r, param)
	if v == "" {
		return def
	}
	value, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		return def
	}
	return utils.ToTime(value)
}

func RequiredParamInt(r *http.Request, param string, msg string) (int, error) {
	p := ParamString(r, param)
	if p == "" {
		return 0, NewError(responses.ErrorMissingParameter, msg)
	}
	return ParamInt(r, param, 0), nil
}

func ParamInt(r *http.Request, param string, def int) int {
	v := ParamString(r, param)
	if v == "" {
		return def
	}
	value, err := strconv.ParseInt(v, 10, 32)
	if err != nil {
		return def
	}
	return int(value)
}

func ParamInts(r *http.Request, param string) []int {
	pStr := ParamStrings(r, param)
	ints := make([]int, 0, len(pStr))
	for _, s := range pStr {
		i, err := strconv.ParseInt(s, 10, 32)
		if err == nil {
			ints = append(ints, int(i))
		}
	}
	return ints
}

func ParamBool(r *http.Request, param string, def bool) bool {
	p := ParamString(r, param)
	if p == "" {
		return def
	}
	return strings.Index("/true/on/1/", "/"+p+"/") != -1
}

type SubsonicError struct {
	code     int
	messages []interface{}
}

func NewError(code int, message ...interface{}) error {
	return SubsonicError{
		code:     code,
		messages: message,
	}
}

func (e SubsonicError) Error() string {
	var msg string
	if len(e.messages) == 0 {
		msg = responses.ErrorMsg(e.code)
	} else {
		msg = fmt.Sprintf(e.messages[0].(string), e.messages[1:]...)
	}
	return msg
}

func ToAlbums(entries engine.Entries) []responses.Child {
	children := make([]responses.Child, len(entries))
	for i, entry := range entries {
		children[i] = ToAlbum(entry)
	}
	return children
}

func ToAlbum(entry engine.Entry) responses.Child {
	album := ToChild(entry)
	album.Name = album.Title
	album.Title = ""
	album.Parent = ""
	album.Album = ""
	album.AlbumId = ""
	return album
}

func ToChildren(entries engine.Entries) []responses.Child {
	children := make([]responses.Child, len(entries))
	for i, entry := range entries {
		children[i] = ToChild(entry)
	}
	return children
}

func ToChild(entry engine.Entry) responses.Child {
	child := responses.Child{}
	child.Id = entry.Id
	child.Title = entry.Title
	child.IsDir = entry.IsDir
	child.Parent = entry.Parent
	child.Album = entry.Album
	child.Year = entry.Year
	child.Artist = entry.Artist
	child.Genre = entry.Genre
	child.CoverArt = entry.CoverArt
	child.Track = entry.Track
	child.Duration = entry.Duration
	child.Size = entry.Size
	child.Suffix = entry.Suffix
	child.BitRate = entry.BitRate
	child.ContentType = entry.ContentType
	if !entry.Starred.IsZero() {
		child.Starred = &entry.Starred
	}
	child.Path = entry.Path
	child.PlayCount = entry.PlayCount
	child.DiscNumber = entry.DiscNumber
	if !entry.Created.IsZero() {
		child.Created = &entry.Created
	}
	child.AlbumId = entry.AlbumId
	child.ArtistId = entry.ArtistId
	child.Type = entry.Type
	child.UserRating = entry.UserRating
	child.SongCount = entry.SongCount
	return child
}
