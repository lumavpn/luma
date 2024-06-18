package errors

import (
	"context"
	"errors"
	"io"
	"net"
	"os"
	"syscall"

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

type extendedError struct {
	message string
	cause   error
}

func (e *extendedError) Error() string {
	if e.cause == nil {
		return e.message
	}
	return e.cause.Error() + ": " + e.message
}

func (e *extendedError) Unwrap() error {
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

func Extend(cause error, message ...any) error {
	if cause == nil {
		panic("extend on an nil error")
	}
	return &extendedError{util.ToString(message...), cause}
}

func IsClosedOrCanceled(err error) bool {
	return IsMulti(err, io.EOF, net.ErrClosed, io.ErrClosedPipe, os.ErrClosed, syscall.EPIPE, syscall.ECONNRESET, context.Canceled, context.DeadlineExceeded)
}

func IsClosed(err error) bool {
	return IsMulti(err, io.EOF, net.ErrClosed, io.ErrClosedPipe, os.ErrClosed, syscall.EPIPE, syscall.ECONNRESET)
}

func IsCanceled(err error) bool {
	return IsMulti(err, context.Canceled, context.DeadlineExceeded)
}
