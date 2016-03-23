package itunesbridge

import (
	"fmt"
	"time"
)

type ItunesControl interface {
	MarkAsPlayed(trackId string, playDate time.Time) error
	MarkAsSkipped(trackId string, skipDate time.Time) error
	SetTrackLoved(trackId string, loved bool) error
	SetAlbumLoved(trackId string, loved bool) error
	SetTrackRating(trackId string, rating int) error
	SetAlbumRating(trackId string, rating int) error
}

func NewItunesControl() ItunesControl {
	return &itunesControl{}
}

type itunesControl struct{}

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
