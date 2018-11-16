package wire

import (
	"github.com/pkg/errors"
)

var (
	ErrUnauthorized           = errors.New("Not authorized to pull object")
	ErrObjectNotFound         = errors.New("Object not found")
	ErrTerminatedWhileSending = errors.New("Terminated While Sending")
	ErrProtocol               = errors.New("Protocol error")
)
