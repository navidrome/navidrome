package criteria

import (
	"encoding/json"
	"fmt"
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

func unmarshalExpression(opName string, rawValue json.RawMessage) Expression {
	m := make(map[string]any)
	err := json.Unmarshal(rawValue, &m)
	if err != nil {
		return nil
	}
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
	}
	return nil
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
