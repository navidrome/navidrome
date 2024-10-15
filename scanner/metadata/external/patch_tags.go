package external

import (
	"github.com/navidrome/navidrome/scanner/metadata"
)

func PatchTags(filePath string, parsedTags metadata.ParsedTags, externalTags []Tags) metadata.ParsedTags {
	setTags := metadata.ParsedTags{}
	removeTags := make(map[string]struct{})
	for _, ext := range externalTags {
		if ext.matchesFile(filePath[len(ext.BasePath)+1:]) {
			for k, v := range ext.SetTags {
				setTags[k] = []string{v}
			}
			for k := range ext.RemoveTags {
				removeTags[k] = struct{}{}
			}
		}
	}
	for k, v := range parsedTags {
		if _, ok := setTags[k]; ok {
			continue
		}
		if _, ok := removeTags[k]; ok {
			continue
		}
		setTags[k] = v
	}
	return setTags
}
