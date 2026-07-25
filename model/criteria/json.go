package criteria

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

type unmarshalConjunctionType []Expression

func (uc *unmarshalConjunctionType) UnmarshalJSON(data []byte) error {
	var raw []map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	var es unmarshalConjunctionType
	for _, e := range raw {
		for k, v := range e {
			k = strings.ToLower(k)
			expr := unmarshalExpression(k, v)
			if expr == nil {
				expr = unmarshalConjunction(k, v)
			}
			if expr == nil {
				return fmt.Errorf(`invalid expression key '%s'`, k)
			}
			es = append(es, expr)
		}
	}
	*uc = es
	return nil
}

// optionalConjunction is a top-level "all"/"any" value that remembers whether its
// key was present at all, so a Criteria providing both can be rejected. encoding/json
// calls UnmarshalJSON even for a JSON null, so present is set whenever the key appears
// — including as [] or null — while an absent key leaves it false.
type optionalConjunction struct {
	present bool
	rules   unmarshalConjunctionType
}

func (o *optionalConjunction) UnmarshalJSON(data []byte) error {
	o.present = true
	return json.Unmarshal(data, &o.rules)
}

func unmarshalExpression(opName string, rawValue json.RawMessage) Expression {
	m := make(map[string]any)
	err := json.Unmarshal(rawValue, &m)
	if err != nil {
		return nil
	}
	normalizeBoolFields(m)
	switch opName {
	case "is":
		return Is(m)
	case "isnot":
		return IsNot(m)
	case "gt":
		return Gt(m)
	case "lt":
		return Lt(m)
	case "contains":
		return Contains(m)
	case "notcontains":
		return NotContains(m)
	case "startswith":
		return StartsWith(m)
	case "endswith":
		return EndsWith(m)
	case "intherange":
		return InTheRange(m)
	case "before":
		return Before(m)
	case "after":
		return After(m)
	case "inthelast":
		return InTheLast(m)
	case "notinthelast":
		return NotInTheLast(m)
	case "inplaylist":
		return InPlaylist(m)
	case "notinplaylist":
		return NotInPlaylist(m)
	case "ismissing":
		normalizeAllBoolFields(m)
		return IsMissing(m)
	case "ispresent":
		normalizeAllBoolFields(m)
		return IsPresent(m)
	}
	return nil
}

func normalizeAllBoolFields(m map[string]any) {
	for k, v := range m {
		m[k] = normalizeBoolValue(v)
	}
}

func normalizeBoolFields(m map[string]any) {
	for field, value := range m {
		info, ok := LookupField(field)
		if ok && info.Boolean {
			m[field] = normalizeBoolValue(value)
		}
	}
}

// ToBool coerces a criteria value to a bool, accepting the forms criteria values take: a real bool,
// a strconv.ParseBool-parseable string, or a JSON number that is exactly 0 or 1. Any other value
// (other numbers, slices, nil, unparseable strings) returns ok=false so callers can handle it.
func ToBool(v any) (bool, bool) {
	switch val := v.(type) {
	case bool:
		return val, true
	case string:
		b, err := strconv.ParseBool(val)
		return b, err == nil
	case float64:
		switch val {
		case 1:
			return true, true
		case 0:
			return false, true
		}
	}
	return false, false
}

// normalizeBoolValue leaves non-boolean values unchanged so they flow through to their own validation.
func normalizeBoolValue(v any) any {
	if b, ok := ToBool(v); ok {
		return b
	}
	return v
}

func unmarshalConjunction(conjName string, rawValue json.RawMessage) Expression {
	var items unmarshalConjunctionType
	err := json.Unmarshal(rawValue, &items)
	if err != nil {
		return nil
	}
	switch conjName {
	case "any":
		return Any(items)
	case "all":
		return All(items)
	}
	return nil
}

func marshalExpression(name string, value map[string]any) ([]byte, error) {
	if len(value) != 1 {
		return nil, fmt.Errorf(`invalid %s expression length %d for values %v`, name, len(value), value)
	}
	b := strings.Builder{}
	b.WriteString(`{"` + name + `":{`)
	for f, v := range value {
		j, err := json.Marshal(v)
		if err != nil {
			return nil, err
		}
		b.WriteString(`"` + f + `":`)
		b.Write(j)
		break
	}
	b.WriteString("}}")
	return []byte(b.String()), nil
}

func marshalConjunction(name string, conj []Expression) ([]byte, error) {
	aux := struct {
		All []Expression `json:"all,omitempty"`
		Any []Expression `json:"any,omitempty"`
	}{}
	if name == "any" {
		aux.Any = conj
	} else {
		aux.All = conj
	}
	return json.Marshal(aux)
}
