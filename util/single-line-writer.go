package util

import (
	"fmt"
	"io"
	"strings"
)

type SingleLineWriter struct {
	out     io.Writer
	prevLen int
}

func NewSingleLineWriter(out io.Writer) *SingleLineWriter {
	return &SingleLineWriter{out: out}
}

func (w *SingleLineWriter) Printf(format string, args ...interface{}) {
	str := fmt.Sprintf(format, args...)
	newLen := len(str)

	if newLen < w.prevLen {
		spaces := strings.Repeat(" ", w.prevLen-newLen)
		str = str + spaces
	}

	fmt.Fprint(w.out, str+"\r")
	w.prevLen = newLen
}
