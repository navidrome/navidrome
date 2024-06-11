package model

import (
	"cmp"
	"crypto/md5"
	"fmt"
	"slices"
	"strings"

	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/utils/slice"
)

type Tag struct {
	ID       string
	TagName  string
	TagValue string
}

type TagList []Tag

func (t Tag) String() string {
	return fmt.Sprintf("%s=%s", t.TagName, t.TagValue)
}

func NewTag(name, value string) Tag {
	name = strings.ToLower(name)
	id := fmt.Sprintf("%x", md5.Sum([]byte(name+consts.Zwsp+strings.ToLower(value))))
	return Tag{
		ID:       id,
		TagName:  name,
		TagValue: value,
	}
}

type Tags map[string][]string

func (t Tags) Values(name string) []string {
	return t[name]
}

func (t Tags) Flatten(name string) TagList {
	var tags TagList
	for _, v := range t[name] {
		tags = append(tags, NewTag(name, v))
	}
	return tags
}

func (t Tags) FlattenAll() TagList {
	var tags TagList
	for name, values := range t {
		for _, v := range values {
			tags = append(tags, NewTag(name, v))
		}
	}
	return tags
}

func (t Tags) Hash() string {
	if len(t) == 0 {
		return ""
	}
	all := t.FlattenAll()
	slices.SortFunc(all, func(a, b Tag) int {
		return cmp.Compare(a.ID, b.ID)
	})
	sum := md5.New()
	for _, tag := range all {
		sum.Write([]byte(tag.ID))
	}
	return fmt.Sprintf("%x", sum.Sum(nil))
}

func (t Tags) ToGenres() (string, Genres) {
	var genres Genres
	for _, g := range t.Values("genre") {
		t := NewTag("genre", g)
		genres = append(genres, Genre{ID: t.ID, Name: g})
	}
	// TODO This will not work, as there is only one instance of each genre in the tags
	return slice.MostFrequent(t.Values("genre")), genres
}

// Merge merges the tags from another Tags object into this one, removing any duplicates
func (t Tags) Merge(tags Tags) {
	for name, values := range tags {
		for _, v := range values {
			t.Add(name, v)
		}
	}
}

func (t Tags) Add(name string, v string) {
	for _, existing := range t[name] {
		if existing == v {
			return
		}
	}
	t[name] = append(t[name], v)
}

type TagRepository interface {
	Add(...Tag) error
}
