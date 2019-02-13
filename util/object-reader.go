package util

import (
	"io"
	"io/ioutil"
)

type ReadAllCloser struct {
	io.Reader
}

func MakeReadAllCloser(r io.Reader) io.ReadCloser {
	rc := ReadAllCloser{
		Reader: r,
	}
	return rc
}

func (rc ReadAllCloser) Close() error {
	_, err := ioutil.ReadAll(rc)
	return err
}
