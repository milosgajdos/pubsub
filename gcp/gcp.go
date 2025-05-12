package gcp

import (
	"errors"
	"time"
)

const (
	DefaultInitRetryBackoff = 250 * time.Millisecond
	DefaultMaxRetryBackoff  = 60 * time.Second
)

var (
	ErrInvalidProject     = errors.New("invalid project")
	ErrInvalidTopic       = errors.New("invalid topic")
	ErrInvalidSubsciption = errors.New("invalid subscription")
	ErrAlreadySubscribed  = errors.New("already subscribed to the topic")
)
