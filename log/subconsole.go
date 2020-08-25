package log

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"strings"

	"github.com/twitchylinux/ccr/vts"
)

type parentConsole interface {
	Stdout() io.Writer
	Stderr() io.Writer
	Operation(key, msg, prefix string) vts.Console
	Done() error
	finishedOperation(key string) error
}

// SubConsole represents a logger for a sub-operation.
type SubConsole struct {
	parentConsole

	out    *bufio.Writer
	err    *bufio.Writer
	key    string
	msg    string
	prefix string
}

func (t *SubConsole) Stdout() io.Writer {
	return &proxyWriter{t.out, t.prefix, true}
}
func (t *SubConsole) Stderr() io.Writer {
	return &proxyWriter{t.err, t.prefix, true}
}
func (t *SubConsole) Done() error {
	t.out.WriteString("\n")
	err := t.out.Flush()
	if err2 := t.err.Flush(); err == nil {
		err = err2
	}
	if err2 := t.parentConsole.finishedOperation(t.key); err == nil {
		err = err2
	}
	return err
}

type proxyWriter struct {
	*bufio.Writer
	name            string
	addPrefixAtNext bool
}

func (w *proxyWriter) Write(b []byte) (int, error) {
	l, s := len(b), string(b)
	pfx := fmt.Sprintf("\n[%s] ", w.name)

	if w.addPrefixAtNext {
		w.Writer.WriteString(pfx)
		w.addPrefixAtNext = false
	}

	if strings.HasSuffix(s, "\n") {
		s = s[:len(s)-1]
		w.addPrefixAtNext = true
	}

	d := bytes.NewReader([]byte(strings.Replace(s, "\n", pfx, -1)))
	if _, err := io.Copy(w.Writer, d); err != nil {
		return 0, err
	}
	return l, w.Writer.Flush()
}
