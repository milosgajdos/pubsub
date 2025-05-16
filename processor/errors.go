package processor

import "errors"

var (
	ErrInvalidMesageType  = errors.New("invalid message type")
	ErrUnknownMessageType = errors.New("unknown message type")
	ErrInvalidHandler     = errors.New("invalid handler")
)
