package domain

import "errors"

var (
	ErrEmptyRules           = errors.New("got empty rules")
	ErrUnsupportedProxyType = errors.New("unsupported proxy type")
	ErrNoResponseChoices    = errors.New("no response choices")
)
