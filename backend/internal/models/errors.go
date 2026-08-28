package models

import "fmt"

type ErrorKind string

const (
	KindInvalid       ErrorKind = "invalid"
	KindNotFound      ErrorKind = "not_found"
	KindConflict      ErrorKind = "conflict"
	KindConflictState ErrorKind = "conflict_state"
	KindInternal      ErrorKind = "internal"
)

type Error struct {
	Kind    ErrorKind
	Message string
}

func (e *Error) Error() string {
	if e == nil {
		return ""
	}
	return e.Message
}

func Invalid(msg string) *Error {
	return &Error{Kind: KindInvalid, Message: msg}
}

func NotFound(msg string) *Error {
	return &Error{Kind: KindNotFound, Message: msg}
}

func Conflict(msg string) *Error {
	return &Error{Kind: KindConflict, Message: msg}
}

func ConflictState(msg string) *Error {
	return &Error{Kind: KindConflictState, Message: msg}
}

func Internal(msg string) *Error {
	return &Error{Kind: KindInternal, Message: msg}
}

func WrapInternal(err error) *Error {
	if err == nil {
		return nil
	}
	var apiErr *Error
	if AsError(err, &apiErr) {
		return apiErr
	}
	return Internal(err.Error())
}

func AsError(err error, target **Error) bool {
	if err == nil {
		return false
	}
	e, ok := err.(*Error)
	if !ok {
		return false
	}
	*target = e
	return true
}

func (e *Error) Format(name string) *Error {
	if e == nil {
		return nil
	}
	return &Error{Kind: e.Kind, Message: fmt.Sprintf(e.Message, name)}
}
