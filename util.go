package main

import (
	"bytes"
	stdErrors "errors"
	"fmt"
	"os"
	"strings"

	"github.com/go-errors/errors"
)

// wrapErrorf attaches additional information to an error. If err is
// of type *errors.Error, the stack is preserved.
func wrapErrorf(err error, format string, a ...interface{}) error {
	// First, create the full error string.
	var s = fmt.Sprintf(format, a...)
	s += ": " + err.Error()
	if e, ok := err.(*errors.Error); ok {
		// Only overwrite the error message.
		e.Err = stdErrors.New(s)
		return err
	}
	return errors.Wrap(err, 1)
}

func wrapError(e interface{}) error {
	return errors.Wrap(e, 1)
}

func formatStackFrames(sfs []errors.StackFrame, fnName string) string {
	wd, err := os.Getwd()
	if err == nil {
		wd += "/"
	}
	buf := &bytes.Buffer{}
	for _, sf := range sfs {
		if sf.Name == fnName {
			break
		}
		f := strings.TrimPrefix(sf.File, wd)
		fmt.Fprintf(buf, "%s:%d:%s", f, sf.LineNumber, sf.Name)
		if source, err := sf.SourceLine(); err != nil {
			buf.WriteString("\n")
		} else {
			fmt.Fprintf(buf, "\n\t%s\n", source)
		}
	}
	return buf.String()
}
