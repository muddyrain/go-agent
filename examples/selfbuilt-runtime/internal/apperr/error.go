package apperr

import "errors"

type Code string

const (
	CodeInternal Code = "INTERNAL"
	CodeConfig   Code = "CONFIG_ERROR"
)

type Error struct {
	Code    Code
	Message string
	Err     error
}

func (e *Error) Error() string {
	if e.Err == nil {
		return e.Message
	}

	return e.Message + ": " + e.Err.Error()
}

func (e *Error) Unwrap() error {
	return e.Err
}

func New(code Code, message string) error {
	return &Error{
		Code:    code,
		Message: message,
	}
}

func Wrap(code Code, message string, err error) error {
	if err == nil {
		return nil
	}

	return &Error{
		Code:    code,
		Message: message,
		Err:     err,
	}
}

func CodeOf(err error) Code {
	var appErr *Error

	if errors.As(err, &appErr) {
		return appErr.Code
	}

	return CodeInternal
}
