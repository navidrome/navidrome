package criteria

import (
	"encoding/json"
	"fmt"
)

// Group is a nested all/any rule group with its own selection options. Groups
// without metadata are still represented by bare All/Any values when parsed.
type Group struct {
	Expression
	Sort         string
	Order        string
	Limit        int
	LimitPercent int
	Offset       int
}

func (Group) fields() map[string]any { return nil }

func (g Group) hasMetadata() bool {
	return g.Sort != "" || g.Order != "" || g.Limit != 0 || g.LimitPercent != 0 || g.Offset != 0
}

func (g Group) MarshalJSON() ([]byte, error) {
	var key string
	var rules []Expression
	switch e := g.Expression.(type) {
	case All:
		key, rules = "all", e
	case Any:
		key, rules = "any", e
	default:
		return nil, fmt.Errorf("invalid group expression type %T", g.Expression)
	}
	if !g.hasMetadata() {
		return marshalConjunction(key, rules)
	}

	obj := map[string]any{key: rules}
	if g.Sort != "" {
		obj["sort"] = g.Sort
	}
	if g.Order != "" {
		obj["order"] = g.Order
	}
	if g.Limit != 0 {
		obj["limit"] = g.Limit
	}
	if g.LimitPercent != 0 {
		obj["limitPercent"] = g.LimitPercent
	}
	if g.Offset != 0 {
		obj["offset"] = g.Offset
	}
	return json.Marshal(obj)
}

func (g Group) ChildPlaylistIds() []string {
	return extractPlaylistField(g.Expression, "id")
}

func (g Group) ChildPlaylistPaths() []string {
	return extractPlaylistField(g.Expression, "path")
}

func (g Group) EffectiveLimit(totalCount int64) int {
	return effectiveLimit(g.Limit, g.LimitPercent, totalCount)
}

func (g Group) IsPercentageLimit() bool {
	return isPercentageLimit(g.Limit, g.LimitPercent)
}

func (g Group) OrderByFields(parent Criteria) []SortField {
	sortValue := g.Sort
	order := g.Order
	if sortValue == "" {
		sortValue = parent.Sort
		if order == "" {
			order = parent.Order
		}
	}
	return parseSortFields(sortValue, order)
}
