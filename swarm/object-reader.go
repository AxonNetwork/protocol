package swarm

import (
	"io"

	gitplumbing "gopkg.in/src-d/go-git.v4/plumbing"
)

type ObjectReader interface {
	io.ReadCloser
	Len() int64
	Type() gitplumbing.ObjectType
}

type objectReader struct {
	io.Reader
	io.Closer
	objectLen  int64
	objectType gitplumbing.ObjectType
}

func (or objectReader) Len() int64 {
	return or.objectLen
}

func (or objectReader) Type() gitplumbing.ObjectType {
	return or.objectType
}
