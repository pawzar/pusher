package pusher

import (
	"errors"
)

var NotificationError = errors.New("notification problem")

type Error struct {
	Msg string
	Err error
}

func (e *Error) Unwrap() error { return NotificationError }
func (e *Error) Error() string { return e.Err.Error() }
