package errors

import "fmt"

type GinitError struct {
	Msg      string
	Provider string
	Err      error
	// Hint tells the user how to fix the problem. Error() always includes it.
	Hint string
}

func (e *GinitError) Error() string {
	prefix := ""
	if e.Provider != "" {
		prefix = "[" + e.Provider + "] "
	}
	msg := prefix + e.Msg
	if e.Err != nil {
		msg = fmt.Sprintf("%s: %v", msg, e.Err)
	}
	if e.Hint != "" {
		msg += "; " + e.Hint
	}
	return msg
}

func (e *GinitError) Unwrap() error {
	return e.Err
}

func New(msg string, err error) error {
	return &GinitError{Msg: msg, Err: err}
}

func NewProvider(provider, msg string, err error) error {
	return &GinitError{Msg: msg, Provider: provider, Err: err}
}

// NewHint creates an error that carries a fix for the user in its message.
func NewHint(msg, hint string, err error) error {
	return &GinitError{Msg: msg, Err: err, Hint: hint}
}
