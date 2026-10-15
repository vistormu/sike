package model

import (
	"errors"
	"fmt"
)

var (
	// ErrNotFound: the thing asked for does not exist.
	ErrNotFound = errors.New("not found")
	// ErrInvalid matches every validation error: errors.Is(err, ErrInvalid).
	ErrInvalid = errors.New("invalid")
	// ErrTaken: a unique value (a user's name) is already in use.
	ErrTaken = errors.New("already taken")
)

// InvalidError is a validation error whose message is written for people: the
// web shows it in a toast (decisions/0007), the terminal prints it.
type InvalidError struct{ Msg string }

func (e InvalidError) Error() string        { return e.Msg }
func (e InvalidError) Is(target error) bool { return target == ErrInvalid }

// Invalid builds an InvalidError, fmt.Sprintf style.
func Invalid(format string, args ...any) error {
	return InvalidError{Msg: fmt.Sprintf(format, args...)}
}
