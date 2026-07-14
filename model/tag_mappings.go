package model

import (
	"cmp"
	"maps"
	"regexp"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"unicode"
	"unicode/utf8"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model/criteria"
	"github.com/navidrome/navidrome/resources"
	"gopkg.in/yaml.v3"
)

type mappingsConf struct {
	Main       tagMappings `yaml:"main"`
	Additional tagMappings `yaml:"additional"`
	Roles      TagConf     `yaml:"roles"`
	Artists    TagConf     `yaml:"artists"`
}

type tagMappings map[TagName]TagConf

type TagConf struct {
	Aliases      []string       `yaml:"aliases"`
	Type         TagType        `yaml:"type"`
	MaxLength    int            `yaml:"maxLength"`
	Split        []string       `yaml:"split"`
	Album        bool           `yaml:"album"`
	SplitRx      *regexp.Regexp `yaml:"-"`
	ExceptionsRx *regexp.Regexp `yaml:"-"`
}

// SplitTagValue splits tag values by the configured split separators.
// Each value in the input slice is individually split and trimmed.
func (c TagConf) SplitTagValue(values []string) []string {
	if c.SplitRx == nil || len(values) == 0 {
		return values
	}

	var result []string
	for _, tag := range values {
		result = append(result, c.splitValue(tag)...)
	}
	return result
}

func (c TagConf) splitValue(tag string) []string {
	protected := protectedSpans(tag, c.ExceptionsRx)
	var parts []string
	start := 0
	for _, sep := range c.SplitRx.FindAllStringIndex(tag, -1) {
		if overlapsAny(sep, protected) {
			continue
		}
		parts = append(parts, strings.TrimSpace(tag[start:sep[0]]))
		start = sep[1]
	}
	return append(parts, strings.TrimSpace(tag[start:]))
}

// protectedSpans returns the spans of rx matches that sit on word boundaries.
// Boundaries are checked here, rune-aware, because RE2's \b is ASCII-only and
// would silently never match names starting/ending with accented letters.
func protectedSpans(tag string, rx *regexp.Regexp) [][]int {
	if rx == nil {
		return nil
	}
	var spans [][]int
	for _, span := range rx.FindAllStringIndex(tag, -1) {
		if isWordBounded(tag, span[0], span[1]) {
			spans = append(spans, span)
		}
	}
	return spans
}

func isWordBounded(s string, start, end int) bool {
	isWord := func(r rune) bool { return unicode.IsLetter(r) || unicode.IsDigit(r) }
	before, _ := utf8.DecodeLastRuneInString(s[:start])
	after, _ := utf8.DecodeRuneInString(s[end:])
	return !isWord(before) && !isWord(after)
}

func overlapsAny(span []int, spans [][]int) bool {
	for _, s := range spans {
		if span[0] < s[1] && s[0] < span[1] {
			return true
		}
	}
	return false
}

// compileExceptionsRegex builds a case-insensitive regex matching any of the
// given literal names, or nil if there are none.
func compileExceptionsRegex(exceptions []string) *regexp.Regexp {
	var names []string
	for _, e := range exceptions {
		if e = strings.TrimSpace(e); e != "" {
			names = append(names, e)
		}
	}
	if len(names) == 0 {
		return nil
	}
	// Longest-first: Go regex alternation is leftmost-first, so with overlapping
	// entries (e.g. "Iron and Wine Duo" vs "Iron and Wine") the longer name must
	// come first to win. Ties broken lexicographically for determinism.
	slices.SortFunc(names, func(a, b string) int {
		if c := cmp.Compare(len(b), len(a)); c != 0 {
			return c
		}
		return cmp.Compare(a, b)
	})
	escaped := make([]string, len(names))
	for i, name := range names {
		escaped[i] = regexp.QuoteMeta(name)
	}
	rx, err := regexp.Compile("(?i)(" + strings.Join(escaped, "|") + ")")
	if err != nil {
		log.Warn("Error compiling split exceptions regexp", "exceptions", exceptions, err)
		return nil
	}
	return rx
}

type artistSplitExceptionsCache struct {
	names []string
	rx    *regexp.Regexp
}

var artistSplitExceptions atomic.Pointer[artistSplitExceptionsCache]

// artistSplitExceptionsRx returns the regex for Scanner.ArtistSplitExceptions,
// or nil if none are configured. Compiled lazily (config hooks only run once
// per process, before tests can override the option) and cached until the
// configured list changes. Lock-free on the cache-hit path, as this is called
// per tag mapping per scanned file, across concurrent scanner goroutines.
func artistSplitExceptionsRx() *regexp.Regexp {
	names := conf.Server.Scanner.ArtistSplitExceptions
	if c := artistSplitExceptions.Load(); c != nil && slices.Equal(c.names, names) {
		return c.rx
	}
	c := &artistSplitExceptionsCache{names: slices.Clone(names), rx: compileExceptionsRegex(names)}
	artistSplitExceptions.Store(c)
	return c.rx
}

// participantTagNames are the tags that hold artist names (or their sort
// values), where split exceptions apply.
var participantTagNames = sync.OnceValue(func() map[TagName]struct{} {
	names := []TagName{
		TagTrackArtist, TagTrackArtists, TagTrackArtistSort, TagTrackArtistsSort,
		TagAlbumArtist, TagAlbumArtists, TagAlbumArtistSort, TagAlbumArtistsSort,
	}
	set := make(map[TagName]struct{}, len(names)+2*len(AllRoles))
	for _, n := range names {
		set[n] = struct{}{}
	}
	for role := range AllRoles {
		set[TagName(role)] = struct{}{}
		set[TagName(role+"sort")] = struct{}{}
	}
	return set
})

// WithParticipantExceptions returns the conf with the global artist split
// exceptions attached when name is a participant (artist/role) tag.
func (c TagConf) WithParticipantExceptions(name TagName) TagConf {
	if _, ok := participantTagNames()[name]; ok {
		c.ExceptionsRx = artistSplitExceptionsRx()
	}
	return c
}

type TagType string

const (
	TagTypeString  TagType = "string"
	TagTypeInteger TagType = "int"
	TagTypeFloat   TagType = "float"
	TagTypeDate    TagType = "date"
	TagTypeUUID    TagType = "uuid"
	TagTypePair    TagType = "pair"
)

func TagMappings() map[TagName]TagConf {
	mappings, _ := parseMappings()
	return mappings
}

func TagRolesConf() TagConf {
	_, cfg := parseMappings()
	return cfg.Roles
}

func TagArtistsConf() TagConf {
	_, cfg := parseMappings()
	return cfg.Artists
}

func TagMainMappings() map[TagName]TagConf {
	_, mappings := parseMappings()
	return mappings.Main
}

var _mappings mappingsConf

var parseMappings = sync.OnceValues(func() (map[TagName]TagConf, mappingsConf) {
	_mappings.Artists.SplitRx = compileSplitRegex("artists", _mappings.Artists.Split)
	_mappings.Roles.SplitRx = compileSplitRegex("roles", _mappings.Roles.Split)

	normalized := tagMappings{}
	collectTags(_mappings.Main, normalized)
	_mappings.Main = normalized

	normalized = tagMappings{}
	collectTags(_mappings.Additional, normalized)
	_mappings.Additional = normalized

	// Merge main and additional mappings, log an error if a tag is found in both
	for k, v := range _mappings.Main {
		if _, ok := _mappings.Additional[k]; ok {
			log.Error("Tag found in both main and additional mappings", "tag", k)
		}
		normalized[k] = v
	}
	return normalized, _mappings
})

func collectTags(tagMappings, normalized map[TagName]TagConf) {
	for k, v := range tagMappings {
		var aliases []string
		for _, val := range v.Aliases {
			aliases = append(aliases, strings.ToLower(val))
		}
		if v.Split != nil {
			if v.Type != "" && v.Type != TagTypeString {
				log.Error("Tag splitting only available for string types", "tag", k, "split", v.Split,
					"type", string(v.Type))
				v.Split = nil
			} else {
				v.SplitRx = compileSplitRegex(k, v.Split)
			}
		}
		v.Aliases = aliases
		normalized[k.ToLower()] = v
	}
}

func compileSplitRegex(tagName TagName, split []string) *regexp.Regexp {
	// Build a list of escaped, non-empty separators.
	var escaped []string
	for _, s := range split {
		if s == "" {
			continue
		}
		escaped = append(escaped, regexp.QuoteMeta(s))
	}
	// If no valid separators remain, return the original value.
	if len(escaped) == 0 {
		if len(split) > 0 {
			log.Warn("No valid separators found in split list", "split", split, "tag", tagName)
		}
		return nil
	}

	// Create one regex that matches any of the separators (case-insensitive).
	pattern := "(?i)(" + strings.Join(escaped, "|") + ")"
	re, err := regexp.Compile(pattern)
	if err != nil {
		log.Warn("Error compiling regexp for split list", "pattern", pattern, "tag", tagName, "split", split, err)
		return nil
	}
	return re
}

func tagNames() []string {
	mappings := TagMappings()
	names := make([]string, 0, len(mappings))
	for k := range mappings {
		names = append(names, string(k))
	}
	return names
}

func numericTagNames() []string {
	mappings := TagMappings()
	names := make([]string, 0)
	for k, cfg := range mappings {
		if cfg.Type == TagTypeInteger || cfg.Type == TagTypeFloat {
			names = append(names, string(k))
		}
	}
	return names
}

func loadTagMappings() {
	mappingsFile, err := resources.FS().Open("mappings.yaml")
	if err != nil {
		log.Error("Error opening mappings.yaml", err)
	}
	decoder := yaml.NewDecoder(mappingsFile)
	err = decoder.Decode(&_mappings)
	if err != nil {
		log.Error("Error decoding mappings.yaml", err)
	}
	if len(_mappings.Main) == 0 {
		log.Error("No tag mappings found in mappings.yaml, check the format")
	}

	// Use Scanner.GenreSeparators if specified and Tags.genre is not defined
	if conf.Server.Scanner.GenreSeparators != "" && len(conf.Server.Tags["genre"].Aliases) == 0 {
		genreConf := _mappings.Main[TagName("genre")]
		genreConf.Split = strings.Split(conf.Server.Scanner.GenreSeparators, "")
		genreConf.SplitRx = compileSplitRegex("genre", genreConf.Split)
		_mappings.Main[TagName("genre")] = genreConf
		log.Debug("Loading deprecated list of genre separators", "separators", genreConf.Split)
	}

	// Overwrite the default mappings with the ones from the config
	for tag, cfg := range conf.Server.Tags {
		if cfg.Ignore {
			delete(_mappings.Main, TagName(tag))
			delete(_mappings.Additional, TagName(tag))
			continue
		}
		oldValue, ok := _mappings.Main[TagName(tag)]
		if !ok {
			oldValue = _mappings.Additional[TagName(tag)]
		}
		aliases := cfg.Aliases
		if len(aliases) == 0 {
			aliases = oldValue.Aliases
		}
		split := cfg.Split
		if split == nil {
			split = oldValue.Split
		}
		c := TagConf{
			Aliases:   aliases,
			Split:     split,
			Type:      cmp.Or(TagType(cfg.Type), oldValue.Type),
			MaxLength: cmp.Or(cfg.MaxLength, oldValue.MaxLength),
			Album:     cmp.Or(cfg.Album, oldValue.Album),
		}
		c.SplitRx = compileSplitRegex(TagName(tag), c.Split)
		if _, ok := _mappings.Main[TagName(tag)]; ok {
			_mappings.Main[TagName(tag)] = c
		} else {
			_mappings.Additional[TagName(tag)] = c
		}
	}
}

func init() {
	conf.AddHook(func() {
		loadTagMappings()

		// This is here to avoid cyclic imports. The criteria package needs to know all tag names, so they can be
		// used in smart playlists
		criteria.AddRoles(slices.Collect(maps.Keys(AllRoles)))
		criteria.AddTagNames(tagNames())
		criteria.AddNumericTags(numericTagNames())
	})
}
