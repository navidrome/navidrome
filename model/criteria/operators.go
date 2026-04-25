package criteria

import "time"

type (
	All []Expression
	And = All
)

func (All) criteriaExpression() {}

func (all All) MarshalJSON() ([]byte, error) {
	return marshalConjunction("all", all)
}

func (all All) ChildPlaylistIds() (ids []string) {
	return extractPlaylistIds(all)
}

type (
	Any []Expression
	Or  = Any
)

func (Any) criteriaExpression() {}

func (any Any) MarshalJSON() ([]byte, error) {
	return marshalConjunction("any", any)
}

func (any Any) ChildPlaylistIds() (ids []string) {
	return extractPlaylistIds(any)
}

type Is map[string]any
type Eq = Is

func (Is) criteriaExpression() {}

func (is Is) MarshalJSON() ([]byte, error) {
	return marshalExpression("is", is)
}

type IsNot map[string]any

func (IsNot) criteriaExpression() {}

func (in IsNot) MarshalJSON() ([]byte, error) {
	return marshalExpression("isNot", in)
}

type Gt map[string]any

func (Gt) criteriaExpression() {}

func (gt Gt) MarshalJSON() ([]byte, error) {
	return marshalExpression("gt", gt)
}

type Lt map[string]any

func (Lt) criteriaExpression() {}

func (lt Lt) MarshalJSON() ([]byte, error) {
	return marshalExpression("lt", lt)
}

type Before map[string]any

func (Before) criteriaExpression() {}

func (bf Before) MarshalJSON() ([]byte, error) {
	return marshalExpression("before", bf)
}

type After Gt

func (After) criteriaExpression() {}

func (af After) MarshalJSON() ([]byte, error) {
	return marshalExpression("after", af)
}

type Contains map[string]any

func (Contains) criteriaExpression() {}

func (ct Contains) MarshalJSON() ([]byte, error) {
	return marshalExpression("contains", ct)
}

type NotContains map[string]any

func (NotContains) criteriaExpression() {}

func (nct NotContains) MarshalJSON() ([]byte, error) {
	return marshalExpression("notContains", nct)
}

type StartsWith map[string]any

func (StartsWith) criteriaExpression() {}

func (sw StartsWith) MarshalJSON() ([]byte, error) {
	return marshalExpression("startsWith", sw)
}

type EndsWith map[string]any

func (EndsWith) criteriaExpression() {}

func (sw EndsWith) MarshalJSON() ([]byte, error) {
	return marshalExpression("endsWith", sw)
}

type InTheRange map[string]any

func (InTheRange) criteriaExpression() {}

func (itr InTheRange) MarshalJSON() ([]byte, error) {
	return marshalExpression("inTheRange", itr)
}

type InTheLast map[string]any

func (InTheLast) criteriaExpression() {}

func (itl InTheLast) MarshalJSON() ([]byte, error) {
	return marshalExpression("inTheLast", itl)
}

type NotInTheLast map[string]any

func (NotInTheLast) criteriaExpression() {}

func (nitl NotInTheLast) MarshalJSON() ([]byte, error) {
	return marshalExpression("notInTheLast", nitl)
}

func startOfPeriod(numDays int64, from time.Time) string {
	return from.Add(time.Duration(-24*numDays) * time.Hour).Format("2006-01-02")
}

type InPlaylist map[string]any

func (InPlaylist) criteriaExpression() {}

func (ipl InPlaylist) MarshalJSON() ([]byte, error) {
	return marshalExpression("inPlaylist", ipl)
}

type NotInPlaylist map[string]any

func (NotInPlaylist) criteriaExpression() {}

func (ipl NotInPlaylist) MarshalJSON() ([]byte, error) {
	return marshalExpression("notInPlaylist", ipl)
}

func extractPlaylistIds(inputRule any) (ids []string) {
	var id string
	var ok bool

	switch rule := inputRule.(type) {
	case Any:
		for _, rules := range rule {
			ids = append(ids, extractPlaylistIds(rules)...)
		}
	case All:
		for _, rules := range rule {
			ids = append(ids, extractPlaylistIds(rules)...)
		}
	case InPlaylist:
		if id, ok = rule["id"].(string); ok {
			ids = append(ids, id)
		}
	case NotInPlaylist:
		if id, ok = rule["id"].(string); ok {
			ids = append(ids, id)
		}
	}

	return
}
