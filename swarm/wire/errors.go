package wire

import (
	"github.com/pkg/errors"
)

var (
	ErrUnauthorized   = errors.New("Not authorized to pull object")
	ErrObjectNotFound = errors.New("Object not found")
	ErrProtocol       = errors.New("Protocol error")
)
