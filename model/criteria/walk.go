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

func Fields(expr Expression) map[string]any {
	return expr.fields()
}
