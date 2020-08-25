// Package log implements common functionality for logging progress and
// reporting messages to the user.
package log

import (
	"bufio"
	"bytes"
	"io"
	"os"
	"sync"

	"github.com/twitchylinux/ccr/vts"
)

type MsgCategory string

// Common message categories.
const (
	MsgBadDef             MsgCategory = "bad target definition"
	MsgBadFind            MsgCategory = "not found"
	MsgBadRef             MsgCategory = "invalid reference"
	MsgFailedCheck        MsgCategory = "check failed"
	MsgFailedPrecondition MsgCategory = "failed precondition"
)

type opMsg struct {
	isErr    bool
	category MsgCategory
	msg      string
	err      error
	t        vts.Target
}

type lockingWriter struct {
	io.Writer
	l sync.Mutex
}

func (w *lockingWriter) Write(b []byte) (int, error) {
	w.l.Lock()
	defer w.l.Unlock()
	return w.Writer.Write(b)
}

// Console writes messages to stdout.
type Console struct {
	ops map[string]*SubConsole
	out *lockingWriter
	err *lockingWriter
}

func (t *Console) Error(category MsgCategory, err error) error {
	printErr(category, err)
	return err
}
func (t *Console) Warning(category MsgCategory, message string) {

}
func (t *Console) Info(category MsgCategory, message string) {

}
func (t *Console) IsInteractive() bool {
	return true
}
func (t *Console) Stdout() io.Writer {
	if t.out == nil {
		t.out = &lockingWriter{Writer: os.Stdout}
	}
	return t.out
}
func (t *Console) Stderr() io.Writer {
	if t.err == nil {
		t.err = &lockingWriter{Writer: os.Stderr}
	}
	return t.err
}

func (t *Console) Operation(key, msg, prefix string) vts.Console {
	if t.ops == nil {
		t.ops = make(map[string]*SubConsole, 12)
	}
	s := &SubConsole{parentConsole: t,
		key:    key,
		msg:    msg,
		prefix: prefix,
		out:    bufio.NewWriter(t.Stdout()),
		err:    bufio.NewWriter(t.Stderr()),
	}
	t.ops[key] = s

	io.Copy(t.Stdout(), bytes.NewReader([]byte(msg)))
	return s
}

func (t *Console) Done() error {
	return nil
}

func (t *Console) finishedOperation(key string) error {
	if _, ok := t.ops[key]; !ok {
		return os.ErrNotExist
	}
	delete(t.ops, key)
	return nil
}

// Silent stores messages internally, without writing them.
type Silent struct {
	msgs   []opMsg
	stdout bytes.Buffer
	stderr bytes.Buffer
}

func (t *Silent) Error(category MsgCategory, err error) error {
	t.msgs = append(t.msgs, opMsg{
		isErr:    true,
		category: category,
		err:      err,
	})
	return err
}
func (t *Silent) Warning(category MsgCategory, message string) {
	t.msgs = append(t.msgs, opMsg{
		category: category,
		msg:      message,
	})
}
func (t *Silent) Info(category MsgCategory, message string) {
	// Info messages are not recorded for the silent op tracker.
}

func (t *Silent) IsInteractive() bool {
	return false
}

func (t *Silent) Stdout() io.Writer {
	return &t.stdout
}
func (t *Silent) Stderr() io.Writer {
	return &t.stderr
}
func (t *Silent) Operation(key, kind, name string) vts.Console {
	return t
}
func (t *Silent) Done() error {
	return nil
}
