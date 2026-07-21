package common

import (
	"errors"
)

// ErrUnsupportedActorMethod is returned by custom actors when the invoked method isn't supported
type ErrUnsupportedActorMethod struct {
	Method string
}

func (e ErrUnsupportedActorMethod) Error() string {
	return "method '" + e.Method + "' unsupported for actor invocation"
}

func (e ErrUnsupportedActorMethod) Is(target error) bool {
	// Ignore the field method when checking if an error is of the type ErrUnsupportedActorMethod
	_, ok := errors.AsType[ErrUnsupportedActorMethod](target)
	return ok
}
