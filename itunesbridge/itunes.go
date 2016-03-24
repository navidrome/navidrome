package itunesbridge

import (
	"fmt"
	"strings"
	"time"
)

type ItunesControl interface {
	MarkAsPlayed(trackId string, playDate time.Time) error
	MarkAsSkipped(trackId string, skipDate time.Time) error
	SetTrackLoved(trackId string, loved bool) error
	SetAlbumLoved(trackId string, loved bool) error
	SetTrackRating(trackId string, rating int) error
	SetAlbumRating(trackId string, rating int) error
	CreatePlaylist(name string, ids []string) (string, error)
	UpdatePlaylist(playlistId string, ids []string) error
	RenamePlaylist(playlistId, name string) error
	DeletePlaylist(playlistId string) error
}

func NewItunesControl() ItunesControl {
	return &itunesControl{}
}

type itunesControl struct{}

func (c *itunesControl) CreatePlaylist(name string, ids []string) (string, error) {
	pids := `"` + strings.Join(ids, `","`) + `"`
	script := Script{
		fmt.Sprintf(`set pls to (make new user playlist with properties {name:"%s"})`, name),
		fmt.Sprintf(`set pids to {%s}`, pids),
		`repeat with trackPID in pids`,
		`	set myTrack to the first item of (every track whose persistent ID is equal to trackPID)`,
		`	duplicate myTrack to pls`,
		`end repeat`,
		`persistent ID of pls`}
	pid, err := script.OutputString()
	if err != nil {
		return "", err
	}
	return strings.TrimSuffix(pid, "\n"), nil
}

func (c *itunesControl) UpdatePlaylist(playlistId string, ids []string) error {
	pids := `"` + strings.Join(ids, `","`) + `"`
	script := Script{
		fmt.Sprintf(`set pls to the first item of (every playlist whose persistent ID is equal to "%s")`, playlistId),
		`delete every track of pls`,
		fmt.Sprintf(`set pids to {%s}`, pids),
		`repeat with trackPID in pids`,
		`	set myTrack to the first item of (every track whose persistent ID is equal to trackPID)`,
		`	duplicate myTrack to pls`,
		`end repeat`}
	return script.Run()
}

func (c *itunesControl) RenamePlaylist(playlistId, name string) error {
	script := Script{
		fmt.Sprintf(`set pls to the first item of (every playlist whose persistent ID is equal to "%s")`, playlistId),
		`tell pls`,
		fmt.Sprintf(`set name to "%s"`, name),
		`end tell`}
	return script.Run()
}

func (c *itunesControl) DeletePlaylist(playlistId string) error {
	script := Script{
		fmt.Sprintf(`set pls to the first item of (every playlist whose persistent ID is equal to "%s")`, playlistId),
		`delete pls`,
	}
	return script.Run()
}

func (c *itunesControl) MarkAsPlayed(trackId string, playDate time.Time) error {
	script := Script{fmt.Sprintf(
		`set theTrack to the first item of (every track whose persistent ID is equal to "%s")`, trackId),
		`set c to (get played count of theTrack)`,
		`tell theTrack`,
		`set played count to c + 1`,
		fmt.Sprintf(`set played date to date("%s")`, c.formatDateTime(playDate)),
		`end tell`}
	return script.Run()
}

func (c *itunesControl) MarkAsSkipped(trackId string, skipDate time.Time) error {
	script := Script{fmt.Sprintf(
		`set theTrack to the first item of (every track whose persistent ID is equal to "%s")`, trackId),
		`set c to (get skipped count of theTrack)`,
		`tell theTrack`,
		`set skipped count to c + 1`,
		fmt.Sprintf(`set skipped date to date("%s")`, c.formatDateTime(skipDate)),
		`end tell`}
	return script.Run()
}

func (c *itunesControl) SetTrackLoved(trackId string, loved bool) error {
	script := Script{fmt.Sprintf(
		`set theTrack to the first item of (every track whose persistent ID is equal to "%s")`, trackId),
		`tell theTrack`,
		fmt.Sprintf(`set loved to %v`, loved),
		`end tell`}
	return script.Run()
}

func (c *itunesControl) SetAlbumLoved(trackId string, loved bool) error {
	script := Script{fmt.Sprintf(
		`set theTrack to the first item of (every track whose persistent ID is equal to "%s")`, trackId),
		`tell theTrack`,
		fmt.Sprintf(`set album loved to %v`, loved),
		`end tell`}
	return script.Run()
}

func (c *itunesControl) SetTrackRating(trackId string, rating int) error {
	script := Script{fmt.Sprintf(
		`set theTrack to the first item of (every track whose persistent ID is equal to "%s")`, trackId),
		`tell theTrack`,
		fmt.Sprintf(`set rating to %d`, rating),
		`end tell`}
	return script.Run()
}

func (c *itunesControl) SetAlbumRating(trackId string, rating int) error {
	script := Script{fmt.Sprintf(
		`set theTrack to the first item of (every track whose persistent ID is equal to "%s")`, trackId),
		`tell theTrack`,
		fmt.Sprintf(`set album rating to %d`, rating),
		`end tell`}
	return script.Run()
}

func (c *itunesControl) formatDateTime(d time.Time) string {
	return d.Format("Jan _2, 2006 3:04PM")
}
