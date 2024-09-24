package external

import (
	"github.com/mitchellh/mapstructure"
	"gopkg.in/yaml.v3"
	"strconv"
	"strings"
)

func unmarshalTags(basePath string, bytes []byte) []Tags {
	var raw map[string]any
	if err := yaml.Unmarshal(bytes, &raw); err != nil {
		return nil
	}
	if tags, ok := parseSimpleMap(basePath, "", raw); ok {
		return []Tags{tags}
	}
	if tags, ok := parseFullEntry(basePath, "", raw); ok {
		return []Tags{tags}
	}
	var patternMap map[string]any
	if mapstructure.Decode(raw, &patternMap) != nil || len(patternMap) == 0 {
		return nil
	}
	tagEntries := make([]Tags, 0, len(patternMap))
	for pattern, rawEntry := range patternMap {
		if tags, ok := parseSimpleMap(basePath, pattern, rawEntry); ok {
			tagEntries = append(tagEntries, tags)
			continue
		}
		if tags, ok := parseFullEntry(basePath, pattern, rawEntry); ok {
			tagEntries = append(tagEntries, tags)
			continue
		}
		return nil
	}
	return tagEntries
}

func parseSimpleMap(basePath string, pattern string, raw any) (Tags, bool) {
	var simpleMapAny map[string]any
	if mapstructure.Decode(raw, &simpleMapAny) != nil {
		return Tags{}, false
	}
	if len(simpleMapAny) == 0 {
		return Tags{}, false
	}
	setTags, ok := createSetTags(simpleMapAny)
	if !ok {
		return Tags{}, false
	}
	return Tags{
		BasePath: basePath,
		Pattern:  pattern,
		SetTags:  setTags,
	}, true
}

type rawFullEntry struct {
	SetTags    map[string]any `mapstructure:"setTags"`
	RemoveTags []string       `mapstructure:"removeTags"`
}

func parseFullEntry(basePath string, pattern string, raw any) (Tags, bool) {
	var fullEntry rawFullEntry
	if mapstructure.Decode(raw, &fullEntry) != nil {
		return Tags{}, false
	}
	if len(fullEntry.SetTags) == 0 && len(fullEntry.RemoveTags) == 0 {
		return Tags{}, false
	}
	setTags, ok := createSetTags(fullEntry.SetTags)
	if !ok {
		return Tags{}, false
	}
	removeTags := make(map[string]struct{}, len(fullEntry.RemoveTags))
	for _, tag := range fullEntry.RemoveTags {
		removeTags[normalizeTagKey(tag)] = struct{}{}
	}
	return Tags{
		BasePath:   basePath,
		Pattern:    pattern,
		SetTags:    setTags,
		RemoveTags: removeTags,
	}, true
}

func createSetTags(rawMap map[string]any) (map[string]string, bool) {
	if len(rawMap) == 0 {
		return nil, false
	}
	parsedMap := make(map[string]string, len(rawMap))
	for key, rawValue := range rawMap {
		value, ok := parseTagValue(rawValue)
		if !ok {
			return nil, false
		}
		parsedMap[normalizeTagKey(key)] = value
	}
	return parsedMap, true
}

func parseTagValue(raw any) (string, bool) {
	switch raw := raw.(type) {
	case string:
		return raw, true
	case int:
		return strconv.Itoa(raw), true
	case bool:
		return strconv.FormatBool(raw), true
	}
	return "", false
}

func normalizeTagKey(key string) string {
	return strings.ToLower(key)
}
