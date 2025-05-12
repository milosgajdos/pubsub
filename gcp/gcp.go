package gcp

import "time"

const (
	DefaultInitRetryBackoff = 250 * time.Millisecond
	DefaultMaxRetryBackoff  = 60 * time.Second
)
