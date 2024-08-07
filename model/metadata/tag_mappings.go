package metadata

import (
	"strings"
	"sync"

	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/resources"
	"gopkg.in/yaml.v3"
)

type tagMapping struct {
	Aliases   []string `yaml:"aliases"`
	Type      TagType  `yaml:"type"`
	MaxLength int      `yaml:"maxLength"`
}

type TagType string

const (
	TagTypeInteger TagType = "integer"
	TagTypeFloat   TagType = "float"
	TagTypeDate    TagType = "date"
	TagTypeUUID    TagType = "uuid"
)

var mappings = sync.OnceValue(func() map[model.TagName]tagMapping {
	mappingsFile, err := resources.FS().Open("mappings.yaml")
	if err != nil {
		log.Error("Error opening mappings.yaml", err)
	}
	decoder := yaml.NewDecoder(mappingsFile)
	var mappings map[model.TagName]tagMapping
	err = decoder.Decode(&mappings)
	if err != nil {
		log.Error("Error decoding mappings.yaml", err)
	}
	normalized := map[model.TagName]tagMapping{}
	for k, v := range mappings {
		var aliases []string
		for _, val := range v.Aliases {
			aliases = append(aliases, strings.ToLower(val))
		}
		normalized[k.ToLower()] = tagMapping{Aliases: aliases, Type: v.Type, MaxLength: v.MaxLength}
	}
	return normalized
})
