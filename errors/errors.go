package errors

import "fmt"

type GinitError struct {
	Msg      string
	Provider string
	Err      error
}

func (e *GinitError) Error() string {
	prefix := ""
	if e.Provider != "" {
		prefix = "[" + e.Provider + "] "
	}
	if e.Err != nil {
		return fmt.Sprintf("%s%s: %v", prefix, e.Msg, e.Err)
	}
	return prefix + e.Msg
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
