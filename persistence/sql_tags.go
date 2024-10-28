package persistence

import (
	"encoding/json"

	. "github.com/Masterminds/squirrel"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/utils/slice"
)

// BFR remove this
func (r sqlRepository) withTags(sql SelectBuilder) SelectBuilder {
	return sql
}

type modelWithTags interface {
	tagIDs() []string
	setTags(tagMap map[string]model.Tag)
}

func (r sqlRepository) loadTags(m modelWithTags) error {
	tagIDs := m.tagIDs()
	if len(tagIDs) == 0 {
		return nil
	}
	query := Select("*").From("tag").Where(Eq{"id": tagIDs})
	var tags model.TagList
	err := r.queryAll(query, &tags)
	if err != nil {
		return err
	}
	tagMap := slice.ToMap(tags, func(t model.Tag) (string, model.Tag) {
		return t.ID, t
	})
	m.setTags(tagMap)
	return nil
}

func buildTagIDs(tags model.Tags) string {
	ids := tags.IDs()
	if len(ids) == 0 {
		return "[]"
	}
	res, _ := json.Marshal(ids)
	return string(res)
}

func tagIDFilter(_ string, idValue any) Sqlizer {
	// We just need to search for the tag.id, as it is calculated based on the tag name and value combined.
	return Like{"tag_ids": `%"` + idValue.(string) + `"%`}
}
