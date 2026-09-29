package model

import (
	"cmp"
	"path/filepath"
	"slices"
	"strings"
)

// LibraryMatcher finds the library that contains an absolute path.
type LibraryMatcher struct {
	libraries    Libraries
	cleanedPaths []string
}

// NewLibraryMatcher sorts the libraries longest path first, so /music-classical is checked before /music.
func NewLibraryMatcher(libs Libraries) *LibraryMatcher {
	libs = slices.Clone(libs)
	slices.SortFunc(libs, func(i, j Library) int {
		return cmp.Compare(len(j.Path), len(i.Path))
	})
	cleanedPaths := make([]string, len(libs))
	for i, lib := range libs {
		cleanedPaths[i] = filepath.Clean(lib.Path)
	}
	return &LibraryMatcher{libraries: libs, cleanedPaths: cleanedPaths}
}

// FindLibrary returns the library whose path contains absolutePath.
func (lm *LibraryMatcher) FindLibrary(absolutePath string) (Library, bool) {
	for i, libPath := range lm.cleanedPaths {
		if strings.HasPrefix(absolutePath, libPath) &&
			(len(absolutePath) == len(libPath) || absolutePath[len(libPath)] == filepath.Separator) {
			return lm.libraries[i], true
		}
	}
	return Library{}, false
}
