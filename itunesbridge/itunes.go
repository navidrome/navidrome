package itunesbridge

import (
	"fmt"
	"time"
)

type ItunesControl interface {
	MarkAsPlayed(id string, playDate time.Time) error
	MarkAsSkipped(id string, skipDate time.Time) error
}

func NewItunesControl() ItunesControl {
	return &itunesControl{}
}

type itunesControl struct{}

func (c *itunesControl) MarkAsPlayed(id string, playDate time.Time) error {
	script := Script{fmt.Sprintf(
		`set theTrack to the first item of (every track whose database ID is equal to "%s")`, id),
		`set c to (get played count of theTrack)`,
		`tell theTrack`,
		`set played count to c + 1`,
		fmt.Sprintf(`set played date to date("%s")`, c.formatDateTime(playDate)),
		`end tell`}
	return script.Run()
}

func (c *itunesControl) MarkAsSkipped(id string, skipDate time.Time) error {
	script := Script{fmt.Sprintf(
		`set theTrack to the first item of (every track whose database ID is equal to "%s")`, id),
		`set c to (get skipped count of theTrack)`,
		`tell theTrack`,
		`set skipped count to c + 1`,
		fmt.Sprintf(`set skipped date to date("%s")`, c.formatDateTime(skipDate)),
		`end tell`}
	return script.Run()
}

func (c *itunesControl) formatDateTime(d time.Time) string {
	return d.Format("Jan _2, 2006 3:04PM")
}
