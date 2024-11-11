package metadata

import (
	"strings"
	"sync"

	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/resources"
	"gopkg.in/yaml.v3"
)

type mappingsConf struct {
	Main       tagMappings `yaml:"main"`
	Additional tagMappings `yaml:"additional"`
	Artist     tagConf     `yaml:"artist"`
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
)

var mappings = sync.OnceValue(func() map[model.TagName]tagConf {
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
	normalized := tagMappings{}
	collectTags(mappings.Main, normalized)
	collectTags(mappings.Additional, normalized)
	return normalized
})

func collectTags(tagMappings, normalized map[model.TagName]tagConf) {
	for k, v := range tagMappings {
		var aliases []string
		for _, val := range v.Aliases {
			aliases = append(aliases, strings.ToLower(val))
		}
		normalized[k.ToLower()] = tagConf{Aliases: aliases, Type: v.Type, MaxLength: v.MaxLength, Split: v.Split}
	}
}
