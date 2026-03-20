package scheduler

import (
	"fmt"
	"math/rand/v2"
	"strconv"
	"strings"
	"time"

	"github.com/robfig/cron/v3"
)

var parser = cron.NewParser(
	cron.SecondOptional | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor,
)

func ParseCrontab(spec string) (cron.Schedule, error) {
	if spec == "" {
		return nil, fmt.Errorf("empty spec string")
	}

	if _, err := time.ParseDuration(spec); err == nil {
		spec = "@every " + spec
	}

	if !strings.Contains(spec, "~") {
		return parser.Parse(spec)
	}

	// Handle TZ=/CRON_TZ= prefix
	var loc *time.Location
	if strings.HasPrefix(spec, "TZ=") || strings.HasPrefix(spec, "CRON_TZ=") {
		i := strings.Index(spec, " ")
		if i == -1 {
			return nil, fmt.Errorf("missing spec after timezone")
		}
		eq := strings.Index(spec, "=")
		var err error
		loc, err = time.LoadLocation(spec[eq+1 : i])
		if err != nil {
			return nil, fmt.Errorf("provided bad location %s: %w", spec[eq+1:i], err)
		}
		spec = strings.TrimSpace(spec[i:])
	}

	// @ descriptors cannot contain ~
	if strings.HasPrefix(spec, "@") {
		return nil, fmt.Errorf("random ~ syntax cannot be used with descriptors: %s", spec)
	}

	fields := strings.Fields(spec)
	fields, err := normalizeFields(fields)
	if err != nil {
		return nil, err
	}

	randomFields := make([]randomField, 6)
	substituteFields := make([]string, 6)
	for i, field := range fields {
		if strings.Contains(field, "~") {
			if strings.ContainsAny(field, ",/") {
				return nil, fmt.Errorf("random ~ cannot be combined with lists or steps: %s", field)
			}
			rf, parseErr := parseRandomField(field, fieldBounds[i])
			if parseErr != nil {
				return nil, parseErr
			}
			randomFields[i] = rf
			substituteFields[i] = fieldDefaults[i]
		} else {
			randomFields[i] = randomField{IsRandom: false}
			substituteFields[i] = field
		}
	}

	substituteSpec := strings.Join(substituteFields, " ")
	baseSched, err := parser.Parse(substituteSpec)
	if err != nil {
		return nil, fmt.Errorf("error parsing non-random fields: %w", err)
	}

	baseSpec, ok := baseSched.(*cron.SpecSchedule)
	if !ok {
		return nil, fmt.Errorf("unexpected schedule type from parser")
	}

	if loc == nil {
		loc = baseSpec.Location
	}

	return &CrontabSchedule{
		Second:   randomFields[0],
		Minute:   randomFields[1],
		Hour:     randomFields[2],
		Dom:      randomFields[3],
		Month:    randomFields[4],
		Dow:      randomFields[5],
		base:     *baseSpec,
		Location: loc,
	}, nil
}

type randomField struct {
	IsRandom bool
	Min, Max uint
}

type CrontabSchedule struct {
	Second, Minute, Hour, Dom, Month, Dow randomField
	base                                  cron.SpecSchedule
	Location                              *time.Location
}

func (s *CrontabSchedule) Next(t time.Time) time.Time {
	resolved := cron.SpecSchedule{
		Second:   s.resolveField(s.Second, s.base.Second),
		Minute:   s.resolveField(s.Minute, s.base.Minute),
		Hour:     s.resolveField(s.Hour, s.base.Hour),
		Dom:      s.resolveField(s.Dom, s.base.Dom),
		Month:    s.resolveField(s.Month, s.base.Month),
		Dow:      s.resolveField(s.Dow, s.base.Dow),
		Location: s.Location,
	}
	return resolved.Next(t)
}

func (s *CrontabSchedule) resolveField(f randomField, baseBits uint64) uint64 {
	if !f.IsRandom {
		return baseBits
	}
	v := f.Min + uint(rand.IntN(int(f.Max-f.Min+1))) //nolint:gosec // Cryptographic randomness not needed for schedule jitter
	return 1 << v
}

type bounds struct {
	min, max uint
}

var fieldBounds = [6]bounds{
	{0, 59}, // Second
	{0, 59}, // Minute
	{0, 23}, // Hour
	{1, 31}, // Dom
	{1, 12}, // Month
	{0, 6},  // Dow
}

var fieldDefaults = [6]string{"0", "0", "0", "1", "1", "0"}

func parseRandomField(field string, b bounds) (randomField, error) {
	parts := strings.SplitN(field, "~", 2)

	min := b.min
	max := b.max

	if parts[0] != "" {
		v, err := strconv.ParseUint(parts[0], 10, 0)
		if err != nil {
			return randomField{}, fmt.Errorf("invalid random range start: %s", parts[0])
		}
		min = uint(v)
	}

	if parts[1] != "" {
		v, err := strconv.ParseUint(parts[1], 10, 0)
		if err != nil {
			return randomField{}, fmt.Errorf("invalid random range end: %s", parts[1])
		}
		max = uint(v)
	}

	if min < b.min {
		return randomField{}, fmt.Errorf("random range start (%d) below minimum (%d): %s", min, b.min, field)
	}
	if max > b.max {
		return randomField{}, fmt.Errorf("random range end (%d) above maximum (%d): %s", max, b.max, field)
	}
	if min > max {
		return randomField{}, fmt.Errorf("random range start (%d) beyond end (%d): %s", min, max, field)
	}

	return randomField{IsRandom: true, Min: min, Max: max}, nil
}

func normalizeFields(fields []string) ([]string, error) {
	switch len(fields) {
	case 5:
		return append([]string{"0"}, fields...), nil
	case 6:
		return fields, nil
	default:
		return nil, fmt.Errorf("expected 5 or 6 fields, found %d: %v", len(fields), fields)
	}
}
