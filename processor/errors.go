package processor

import "errors"

var (
	ErrInvalidMessageType = errors.New("invalid message type")
	ErrUnknownMessageType = errors.New("unknown message type")
	ErrInvalidHandler     = errors.New("invalid handler")
)
