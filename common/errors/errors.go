package errors

import (
	"context"
	"errors"

	"github.com/lumavpn/luma/util"
)

type causeError struct {
	message string
	cause   error
}

func (e *causeError) Error() string {
	return e.message + ": " + e.cause.Error()
}

func (e *causeError) Unwrap() error {
	return e.cause
}

type ErrorHandler interface {
	NewError(ctx context.Context, err error)
}

type MultiError interface {
	Unwrap() []error
}

func New(message ...any) error {
	return errors.New(util.ToString(message...))
}

func Cause(cause error, message ...any) error {
	if cause == nil {
		panic("cause on an nil error")
	}
	return &causeError{util.ToString(message...), cause}
}
