package util

import (
	"io"

	gitplumbing "gopkg.in/src-d/go-git.v4/plumbing"
)

// type ObjectReader interface {
//     io.ReadCloser
//     Len() int64
//     Type() gitplumbing.ObjectType
// }

type ObjectReader struct {
	io.Reader
	io.Closer
	ObjectLen  uint64
	ObjectType gitplumbing.ObjectType
}

func (or ObjectReader) Len() uint64 {
	return or.ObjectLen
}

func (or ObjectReader) Type() gitplumbing.ObjectType {
	return or.ObjectType
}
