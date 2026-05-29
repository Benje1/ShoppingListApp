package logger

import (
	"fmt"
	"runtime"
	"strings"
)

// StackError wraps an error with a stack trace captured at the point WithStack
// is called. Use WithStack when returning errors from service/handler code so
// that logger.Error can print the full trace automatically.
type StackError struct {
	err   error
	stack []uintptr
}

// WithStack wraps err with a stack trace captured at the caller's location.
// If err is nil it returns nil. If err is already a *StackError it is returned
// unchanged so the original capture site is preserved.
func WithStack(err error) error {
	if err == nil {
		return nil
	}
	if _, ok := err.(*StackError); ok {
		return err
	}
	var pcs [32]uintptr
	// skip: runtime.Callers, WithStack, caller of WithStack
	n := runtime.Callers(2, pcs[:])
	return &StackError{err: err, stack: pcs[:n]}
}

func (e *StackError) Error() string { return e.err.Error() }
func (e *StackError) Unwrap() error { return e.err }

// StackTrace returns a formatted, newline-separated stack trace string.
func (e *StackError) StackTrace() string {
	frames := runtime.CallersFrames(e.stack)
	var sb strings.Builder
	for {
		f, more := frames.Next()
		// Skip runtime internals
		if strings.HasPrefix(f.Function, "runtime.") {
			if !more {
				break
			}
			continue
		}
		fmt.Fprintf(&sb, "\n\t%s\n\t\t%s:%d", f.Function, f.File, f.Line)
		if !more {
			break
		}
	}
	return sb.String()
}
