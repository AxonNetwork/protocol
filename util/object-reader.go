package util

import (
	"io"
	"io/ioutil"

	gitplumbing "gopkg.in/src-d/go-git.v4/plumbing"
)

type ReadAllCloser struct {
	r io.Reader
}

func MakeReadAllCloser(r io.Reader) io.ReadCloser {
	rc := ReadAllCloser{
		r: r,
	}
	return rc
}

func (rc ReadAllCloser) Read(p []byte) (n int, err error) {
	return rc.r.Read(p)
}

func (rc ReadAllCloser) Close() error {
	_, err := ioutil.ReadAll(rc.r)
	return err
}

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
