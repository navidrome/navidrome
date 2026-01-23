package model

import (
	"cmp"
	"maps"
	"regexp"
	"slices"
	"strings"
	"sync"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/consts"
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
	Aliases   []string       `yaml:"aliases"`
	Type      TagType        `yaml:"type"`
	MaxLength int            `yaml:"maxLength"`
	Split     []string       `yaml:"split"`
	Album     bool           `yaml:"album"`
	SplitRx   *regexp.Regexp `yaml:"-"`
}

// SplitTagValue splits a tag value by the split separators, but only if it has a single value.
func (c TagConf) SplitTagValue(values []string) []string {
	// If there's not exactly one value or no separators, return early.
	if len(values) != 1 || c.SplitRx == nil {
		return values
	}
	tag := values[0]

	// Replace all occurrences of any separator with the zero-width space.
	tag = c.SplitRx.ReplaceAllString(tag, consts.Zwsp)

	// Split by the zero-width space and trim each substring.
	parts := strings.Split(tag, consts.Zwsp)
	for i, part := range parts {
		parts[i] = strings.TrimSpace(part)
	}
	return parts
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
