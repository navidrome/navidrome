package model

import (
	"cmp"
	"crypto/md5"
	"fmt"
	"slices"
	"strings"

	"github.com/navidrome/navidrome/consts"
)

type Tag struct {
	ID    string
	Name  string
	Value string
}

type TagList []Tag

func (t Tag) String() string {
	return fmt.Sprintf("%s=%s", t.Name, t.Value)
}

func NewTag(name, value string) Tag {
	name = strings.ToLower(name)
	id := fmt.Sprintf("%x", md5.Sum([]byte(name+consts.Zwsp+strings.ToLower(value))))
	return Tag{
		ID:    id,
		Name:  name,
		Value: value,
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

func (t *Tags) Hash() string {
	if len(*t) == 0 {
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

type TagRepository interface {
	Add(...Tag) error
}
