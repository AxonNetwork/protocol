package util

import (
	"io"
)

// CheckClose calls Close on the given io.Closer. If the given *error points to
// nil, it will be assigned the error returned by Close. Otherwise, any error
// returned by Close will be ignored. CheckClose is usually called with defer.
// (Taken from gopkg.in/src-d/go-git.v4)
func CheckClose(c io.Closer, err *error) {
	if cerr := c.Close(); cerr != nil && *err == nil {
		*err = cerr
	}
}

func CheckCloseFunc(closeFn func() error, err *error) {
	if cerr := closeFn(); cerr != nil && *err == nil {
		*err = cerr
	}
}
