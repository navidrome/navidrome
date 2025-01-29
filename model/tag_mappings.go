package model

import (
	"maps"
	"slices"
	"strings"
	"sync"

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
	Aliases   []string `yaml:"aliases"`
	Type      TagType  `yaml:"type"`
	MaxLength int      `yaml:"maxLength"`
	Split     []string `yaml:"split"`
	Album     bool     `yaml:"album"`
}

type TagType string

const (
	TagTypeInteger TagType = "integer"
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
	_, conf := parseMappings()
	return conf.Roles
}

func TagArtistsConf() TagConf {
	_, conf := parseMappings()
	return conf.Artists
}

func TagMainMappings() map[TagName]TagConf {
	_, mappings := parseMappings()
	return mappings.Main
}

var parseMappings = sync.OnceValues(func() (map[TagName]TagConf, mappingsConf) {
	mappingsFile, err := resources.FS().Open("mappings.yaml")
	if err != nil {
		log.Error("Error opening mappings.yaml", err)
	}
	decoder := yaml.NewDecoder(mappingsFile)
	var mappings mappingsConf
	err = decoder.Decode(&mappings)
	if err != nil {
		log.Error("Error decoding mappings.yaml", err)
	}
	if len(mappings.Main) == 0 {
		log.Error("No tag mappings found in mappings.yaml, check the format")
	}

	normalized := tagMappings{}
	collectTags(mappings.Main, normalized)
	mappings.Main = normalized

	normalized = tagMappings{}
	collectTags(mappings.Additional, normalized)
	mappings.Additional = normalized

	// Merge main and additional mappings, log an error if a tag is found in both
	for k, v := range mappings.Main {
		if _, ok := mappings.Additional[k]; ok {
			log.Error("Tag found in both main and additional mappings", "tag", k)
		}
		normalized[k] = v
	}
	return normalized, mappings
})

func collectTags(tagMappings, normalized map[TagName]TagConf) {
	for k, v := range tagMappings {
		var aliases []string
		for _, val := range v.Aliases {
			aliases = append(aliases, strings.ToLower(val))
		}
		if v.Split != nil && v.Type != "" {
			log.Error("Tag splitting only available for string types", "tag", k, "split", v.Split, "type", v.Type)
			v.Split = nil
		}
		v.Aliases = aliases
		normalized[k.ToLower()] = v
	}
}

func tagNames() []string {
	mappings := TagMappings()
	names := make([]string, 0, len(mappings))
	for k := range mappings {
		names = append(names, string(k))
	}
	return names
}

// This is here to avoid cyclic imports. The criteria package needs to know all tag names, so they can be used in
// smart playlists
func init() {
	conf.AddHook(func() {
		criteria.AddRoles(slices.Collect(maps.Keys(AllRoles)))
		criteria.AddTagNames(tagNames())
	})
}
