package persistence

import (
	"encoding/json"
	"fmt"
	"strings"

	. "github.com/Masterminds/squirrel"
	"github.com/navidrome/navidrome/model"
)

// Format of a tag in the DB
type dbTag struct {
	ID    string `json:"id"`
	Value string `json:"value"`
}
type dbTags map[model.TagName][]dbTag

func unmarshalTags(data string) (model.Tags, error) {
	var dbTags dbTags
	err := json.Unmarshal([]byte(data), &dbTags)
	if err != nil {
		return nil, fmt.Errorf("parsing tags: %w", err)
	}

	res := make(model.Tags, len(dbTags))
	for name, tags := range dbTags {
		res[name] = make([]string, len(tags))
		for i, tag := range tags {
			res[name][i] = tag.Value
		}
	}
	return res, nil
}

func marshalTags(tags model.Tags) string {
	dbTags := dbTags{}
	for name, values := range tags {
		for _, value := range values {
			t := model.NewTag(name, value)
			dbTags[name] = append(dbTags[name], dbTag{ID: t.ID, Value: value})
		}
	}
	res, _ := json.Marshal(dbTags)
	return string(res)
}

func tagIDFilter(name string, idValue any) Sqlizer {
	name = strings.TrimSuffix(name, "_id")
	return Exists(
		fmt.Sprintf(`json_tree(tags, "$.%s")`, name),
		And{
			NotEq{"json_tree.atom": nil},
			Eq{"value": idValue},
		},
	)
}
