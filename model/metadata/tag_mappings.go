package metadata

import (
	"maps"
	"slices"
	"strings"
	"sync"

	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/criteria"
	"github.com/navidrome/navidrome/resources"
	"gopkg.in/yaml.v3"
)

type mappingsConf struct {
	Main       tagMappings `yaml:"main"`
	Additional tagMappings `yaml:"additional"`
	Roles      tagConf     `yaml:"roles"`
	Artists    tagConf     `yaml:"artists"`
}

type tagMappings map[model.TagName]tagConf

type tagConf struct {
	Aliases   []string `yaml:"aliases"`
	Type      TagType  `yaml:"type"`
	MaxLength int      `yaml:"maxLength"`
	Split     []string `yaml:"split"`
}

type TagType string

const (
	TagTypeInteger TagType = "integer"
	TagTypeFloat   TagType = "float"
	TagTypeDate    TagType = "date"
	TagTypeUUID    TagType = "uuid"
	TagTypePair    TagType = "pair"
)

func mappings() map[model.TagName]tagConf {
	mappings, _ := parseMappings()
	return mappings
}

func rolesConf() tagConf {
	_, conf := parseMappings()
	return conf.Roles
}

func artistsConf() tagConf {
	_, conf := parseMappings()
	return conf.Artists
}

var parseMappings = sync.OnceValues(func() (map[model.TagName]tagConf, mappingsConf) {
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
	collectTags(mappings.Additional, normalized)
	return normalized, mappings
})

func collectTags(tagMappings, normalized map[model.TagName]tagConf) {
	for k, v := range tagMappings {
		var aliases []string
		for _, val := range v.Aliases {
			aliases = append(aliases, strings.ToLower(val))
		}
		if v.Split != nil && v.Type != "" {
			log.Error("Tag splitting only available for string types", "tag", k, "split", v.Split, "type", v.Type)
			v.Split = nil
		}
		normalized[k.ToLower()] = tagConf{Aliases: aliases, Type: v.Type, MaxLength: v.MaxLength, Split: v.Split}
	}
}

func tagNames() []string {
	mappings := mappings()
	names := make([]string, 0, len(mappings))
	for k := range mappings {
		names = append(names, string(k))
	}
	return names
}

// This is here to avoid cyclic imports. The criteria package needs to know all tag names, so they can be used in
// smart playlists
func init() {
	criteria.AddRoles(slices.Collect(maps.Keys(model.AllRoles)))
	criteria.AddTagNames(tagNames())
}
