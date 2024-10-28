package persistence

import (
	"encoding/json"

	. "github.com/Masterminds/squirrel"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/utils/slice"
)

func (r sqlRepository) withTags(sql SelectBuilder) SelectBuilder {
	return sql
	//return sql.LeftJoin(fmt.Sprintf("item_tags it on it.item_id = %[1]s.id and it.item_type = '%[1]s'", r.tableName)).
	//	LeftJoin("tag on tag.id = it.tag_id").
	//	Columns("json_group_array(distinct(json_object(ifnull(tag.tag_name, ''), tag.tag_value))) as tags")
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

func parseTags(strTags string) (model.Tags, error) {
	tags := model.Tags{}
	if strTags == "" {
		return tags, nil
	}
	var dbTags []map[model.TagName]string
	err := json.Unmarshal([]byte(strTags), &dbTags)
	if err != nil {
		return nil, err
	}
	for _, t := range dbTags {
		for tagName, tagValue := range t {
			if tagName == "" {
				continue
			}
			tags[tagName] = append(tags[tagName], tagValue)
		}
	}
	return tags, nil
}

func (r sqlRepository) updateTags(itemID string, tags model.Tags) error {
	//names := maps.Keys(tags)
	//sqd := Delete("item_tags").Where(And{Eq{"item_id": itemID, "item_type": r.tableName}, NotEq{"tag_name": names}})
	//_, err := r.executeSQL(sqd)
	//if err != nil {
	//	return err
	//}
	//if len(tags) == 0 {
	//	return nil
	//}
	//sqi := Insert("item_tags").Columns("item_id", "item_type", "tag_name", "tag_id").
	//	Suffix("on conflict (item_id, item_type, tag_id) do nothing")
	//for name, values := range tags {
	//	for _, value := range values {
	//		tag := model.NewTag(name, value)
	//		sqi = sqi.Values(itemID, r.tableName, tag.TagName, tag.ID)
	//	}
	//}
	//_, err = r.executeSQL(sqi)
	return nil
}

func tagIDFilter(_ string, idValue any) Sqlizer {
	// We just need to search for the tag.id, as it is calculated based on the tag name and value combined.
	return Like{"tag_ids": "%" + idValue.(string) + "%"}
}
