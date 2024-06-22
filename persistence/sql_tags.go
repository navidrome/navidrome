package persistence

import (
	"encoding/json"
	"fmt"
	"strings"

	. "github.com/Masterminds/squirrel"
	"github.com/navidrome/navidrome/model"
	"golang.org/x/exp/maps"
)

// TODO Consolidate withTags and newSelectWithAnnotation(s)?
func (r sqlRepository) withTags(sql SelectBuilder) SelectBuilder {
	return sql.LeftJoin(fmt.Sprintf("item_tags it on it.item_id = %[1]s.id and it.item_type = '%[1]s'", r.tableName)).
		LeftJoin("tag on tag.id = it.tag_id").
		Columns("json_group_array(distinct(json_object(ifnull(tag.tag_name, ''), tag.tag_value))) as tags")
}

func parseTags(strTags string) (model.Tags, error) {
	tags := model.Tags{}
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
	names := maps.Keys(tags)
	sqd := Delete("item_tags").Where(And{Eq{"item_id": itemID, "item_type": r.tableName}, NotEq{"tag_name": names}})
	_, err := r.executeSQL(sqd)
	if err != nil {
		return err
	}
	if len(tags) == 0 {
		return nil
	}
	sqi := Insert("item_tags").Columns("item_id", "item_type", "tag_name", "tag_id").
		Suffix("on conflict (item_id, item_type, tag_id) do nothing")
	for name, values := range tags {
		for _, value := range values {
			tag := model.NewTag(name, value)
			sqi = sqi.Values(itemID, r.tableName, tag.TagName, tag.ID)
		}
	}
	_, err = r.executeSQL(sqi)
	return err
}

func tagIDFilter(name string, idValue interface{}) Sqlizer {
	tagName := strings.TrimSuffix(name, "_id")
	return Eq{"tag.id": idValue, "tag.tag_name": tagName}
}
