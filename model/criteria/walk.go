package criteria

import "fmt"

type Visitor func(Expression) error

func Walk(expr Expression, visit Visitor) error {
	if expr == nil {
		return nil
	}
	if err := visit(expr); err != nil {
		return err
	}
	switch e := expr.(type) {
	case All:
		for _, child := range e {
			if err := Walk(child, visit); err != nil {
				return err
			}
		}
	case Any:
		for _, child := range e {
			if err := Walk(child, visit); err != nil {
				return err
			}
		}
	case Is, IsNot, Gt, Lt, Before, After, Contains, NotContains, StartsWith, EndsWith, InTheRange, InTheLast, NotInTheLast, InPlaylist, NotInPlaylist:
		return nil
	default:
		return fmt.Errorf("unknown criteria expression type %T", expr)
	}
	return nil
}

// Fields returns field values for leaf expressions only.
// Use Walk to traverse All and Any expressions before calling Fields.
func Fields(expr Expression) map[string]any {
	switch e := expr.(type) {
	case Is:
		return map[string]any(e)
	case IsNot:
		return map[string]any(e)
	case Gt:
		return map[string]any(e)
	case Lt:
		return map[string]any(e)
	case Before:
		return map[string]any(e)
	case After:
		return map[string]any(Gt(e))
	case Contains:
		return map[string]any(e)
	case NotContains:
		return map[string]any(e)
	case StartsWith:
		return map[string]any(e)
	case EndsWith:
		return map[string]any(e)
	case InTheRange:
		return map[string]any(e)
	case InTheLast:
		return map[string]any(e)
	case NotInTheLast:
		return map[string]any(e)
	case InPlaylist:
		return map[string]any(e)
	case NotInPlaylist:
		return map[string]any(e)
	default:
		return nil
	}
}
