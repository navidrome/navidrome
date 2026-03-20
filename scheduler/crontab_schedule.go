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

// ParseCrontab parses a cron expression with support for the crontab(5) random ~ syntax.
// Random values are resolved once at parse time. If no ~ is present, it delegates to
// robfig/cron's standard parser. Duration strings (e.g., "5m") are converted to "@every 5m".
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
	var tzPrefix string
	if strings.HasPrefix(spec, "TZ=") || strings.HasPrefix(spec, "CRON_TZ=") {
		i := strings.Index(spec, " ")
		if i == -1 {
			return nil, fmt.Errorf("missing spec after timezone")
		}
		tzPrefix = spec[:i] + " "
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

	// Resolve each ~ field to a concrete random value
	for i, field := range fields {
		if !strings.Contains(field, "~") {
			continue
		}
		if strings.ContainsAny(field, ",/") {
			return nil, fmt.Errorf("random ~ cannot be combined with lists or steps: %s", field)
		}
		v, parseErr := resolveRandomField(field, fieldBounds[i])
		if parseErr != nil {
			return nil, parseErr
		}
		fields[i] = strconv.FormatUint(uint64(v), 10)
	}

	// Re-assemble and parse with robfig
	resolved := tzPrefix + strings.Join(fields, " ")
	return parser.Parse(resolved)
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

// resolveRandomField parses a ~ field and returns a random value within the range.
func resolveRandomField(field string, b bounds) (uint, error) {
	parts := strings.SplitN(field, "~", 2)

	min := b.min
	max := b.max

	if parts[0] != "" {
		v, err := strconv.ParseUint(parts[0], 10, 0)
		if err != nil {
			return 0, fmt.Errorf("invalid random range start: %s", parts[0])
		}
		min = uint(v)
	}

	if parts[1] != "" {
		v, err := strconv.ParseUint(parts[1], 10, 0)
		if err != nil {
			return 0, fmt.Errorf("invalid random range end: %s", parts[1])
		}
		max = uint(v)
	}

	if min < b.min {
		return 0, fmt.Errorf("random range start (%d) below minimum (%d): %s", min, b.min, field)
	}
	if max > b.max {
		return 0, fmt.Errorf("random range end (%d) above maximum (%d): %s", max, b.max, field)
	}
	if min > max {
		return 0, fmt.Errorf("random range start (%d) beyond end (%d): %s", min, max, field)
	}

	return min + uint(rand.IntN(int(max-min+1))), nil //nolint:gosec // Cryptographic randomness not needed for schedule jitter
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
