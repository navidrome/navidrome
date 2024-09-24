package external

import (
	"os"
	"path/filepath"
)

func ReadTagsFiles(filePaths []string) []Tags {
	var tags []Tags
	for _, filePath := range filePaths {
		tags = append(tags, ReadTagsFile(filePath)...)
	}
	return tags
}

func ReadTagsFile(filePath string) []Tags {
	bytes, err := os.ReadFile(filePath)
	if err != nil {
		return nil
	}
	return unmarshalTags(filepath.Dir(filePath), bytes)
}
