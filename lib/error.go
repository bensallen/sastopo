package sastopo

import "fmt"

type errUnknownType struct{ Msg string }

func (e *errUnknownType) Error() string {
	return fmt.Sprintf("unknown device type: %s", e.Msg)
}
