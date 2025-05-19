package worker

import "errors"

var (
	ErrWorkerRunning    = errors.New("worker already running")
	ErrWorkerNotRunning = errors.New("worker not running")
	ErrNilSubscriber    = errors.New("subscriber cannot be nil")
	ErrNilProcessor     = errors.New("processor cannot be nil")
)
