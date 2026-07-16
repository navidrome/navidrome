package utils

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

func TimeNewest(times ...time.Time) time.Time {
	newest := time.Time{}
	for _, t := range times {
		if t.After(newest) {
			newest = t
		}
	}
	return newest
}

var durationDayWeekRe = regexp.MustCompile(`-?\d+(?:\.\d+)?[dw]`)

// ParseDuration is time.ParseDuration extended with d (24h) and w (168h) units.
// Negative durations are rejected.
func ParseDuration(s string) (time.Duration, error) {
	expanded := durationDayWeekRe.ReplaceAllStringFunc(s, func(match string) string {
		value, err := strconv.ParseFloat(match[:len(match)-1], 64)
		if err != nil {
			return match
		}
		hours := value * 24
		if match[len(match)-1] == 'w' {
			hours = value * 24 * 7
		}
		return strconv.FormatFloat(hours, 'f', -1, 64) + "h"
	})
	d, err := time.ParseDuration(expanded)
	if err != nil {
		return 0, fmt.Errorf("invalid duration %q: %w", s, err)
	}
	if d < 0 {
		return 0, fmt.Errorf("negative duration not allowed: %q", s)
	}
	return d, nil
}

// FormatDuration renders whole w/d multiples with those units, falling back to
// time.Duration.String for the sub-day remainder, so ParseDuration round-trips.
func FormatDuration(d time.Duration) string {
	if d < 24*time.Hour {
		return formatSubDay(d)
	}
	var b strings.Builder
	weekDuration := 7 * 24 * time.Hour
	if weeks := d / weekDuration; weeks > 0 {
		b.WriteString(strconv.Itoa(int(weeks)) + "w")
		d %= weekDuration
	}
	dayDuration := 24 * time.Hour
	if days := d / dayDuration; days > 0 {
		b.WriteString(strconv.Itoa(int(days)) + "d")
		d %= dayDuration
	}
	if d > 0 {
		b.WriteString(formatSubDay(d))
	}
	return b.String()
}

func formatSubDay(d time.Duration) string {
	if d >= time.Hour && d%time.Hour == 0 {
		return strconv.Itoa(int(d/time.Hour)) + "h"
	}
	return d.String()
}
