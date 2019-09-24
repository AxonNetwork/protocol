package util

import (
	"fmt"
	"runtime"
)

type Frame struct {
	Func string
	Line int
	Path string
}

type Frames []Frame

func StackTrace(skip int) Frames {
	var frames Frames

	for {
		pc, path, line, ok := runtime.Caller(skip)
		if !ok {
			break
		}
		fn := runtime.FuncForPC(pc)
		frame := Frame{
			Func: fn.Name(),
			Line: line,
			Path: path,
		}
		frames = append(frames, frame)
		skip++
	}
	return frames
}

func (f Frame) String() string {
	return fmt.Sprintf("%s:%d %s()", f.Path, f.Line, f.Func)
}

func (fs Frames) String() string {
	var s string
	for _, f := range fs {
		s += f.String() + "\n"
	}
	return s
}
