package criteria

import "time"

// Conjunctions need to implement this interface, to allow Criteria to extract child playlist IDs recursively
type conjunction interface {
	ChildPlaylistIds() []string
}

type (
	All []Expression
	And = All
)

func (All) fields() map[string]any { return nil }

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

func (Any) fields() map[string]any { return nil }

func (any Any) MarshalJSON() ([]byte, error) {
	return marshalConjunction("any", any)
}

func (any Any) ChildPlaylistIds() (ids []string) {
	return extractPlaylistIds(any)
}

type Is map[string]any
type Eq = Is

func (is Is) MarshalJSON() ([]byte, error) {
	return marshalExpression("is", is)
}

func (is Is) fields() map[string]any { return is }

type IsNot map[string]any

func (isn IsNot) MarshalJSON() ([]byte, error) {
	return marshalExpression("isNot", isn)
}

func (isn IsNot) fields() map[string]any { return isn }

type Gt map[string]any

func (gt Gt) MarshalJSON() ([]byte, error) {
	return marshalExpression("gt", gt)
}

func (gt Gt) fields() map[string]any { return gt }

type Lt map[string]any

func (lt Lt) MarshalJSON() ([]byte, error) {
	return marshalExpression("lt", lt)
}

func (lt Lt) fields() map[string]any { return lt }

type Before map[string]any

func (bf Before) MarshalJSON() ([]byte, error) {
	return marshalExpression("before", bf)
}

func (bf Before) fields() map[string]any { return bf }

type After Gt

func (af After) MarshalJSON() ([]byte, error) {
	return marshalExpression("after", af)
}

func (af After) fields() map[string]any { return af }

type Contains map[string]any

func (ct Contains) MarshalJSON() ([]byte, error) {
	return marshalExpression("contains", ct)
}

func (ct Contains) fields() map[string]any { return ct }

type NotContains map[string]any

func (nct NotContains) MarshalJSON() ([]byte, error) {
	return marshalExpression("notContains", nct)
}

func (nct NotContains) fields() map[string]any { return nct }

type StartsWith map[string]any

func (sw StartsWith) MarshalJSON() ([]byte, error) {
	return marshalExpression("startsWith", sw)
}

func (sw StartsWith) fields() map[string]any { return sw }

type EndsWith map[string]any

func (ew EndsWith) MarshalJSON() ([]byte, error) {
	return marshalExpression("endsWith", ew)
}

func (ew EndsWith) fields() map[string]any { return ew }

type InTheRange map[string]any

func (itr InTheRange) MarshalJSON() ([]byte, error) {
	return marshalExpression("inTheRange", itr)
}

func (itr InTheRange) fields() map[string]any { return itr }

type InTheLast map[string]any

func (itl InTheLast) MarshalJSON() ([]byte, error) {
	return marshalExpression("inTheLast", itl)
}

func (itl InTheLast) fields() map[string]any { return itl }

type NotInTheLast map[string]any

func (nitl NotInTheLast) MarshalJSON() ([]byte, error) {
	return marshalExpression("notInTheLast", nitl)
}

func (nitl NotInTheLast) fields() map[string]any { return nitl }

func startOfPeriod(numDays int64, from time.Time) string {
	return from.Add(time.Duration(-24*numDays) * time.Hour).Format("2006-01-02")
}

type InPlaylist map[string]any

func (ipl InPlaylist) MarshalJSON() ([]byte, error) {
	return marshalExpression("inPlaylist", ipl)
}

func (ipl InPlaylist) fields() map[string]any { return ipl }

type NotInPlaylist map[string]any

func (nipl NotInPlaylist) MarshalJSON() ([]byte, error) {
	return marshalExpression("notInPlaylist", nipl)
}

func (nipl NotInPlaylist) fields() map[string]any { return nipl }

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
