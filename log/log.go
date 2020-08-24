// Package log implements common functionality for logging progress and
// reporting messages to the user.
package log

import (
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

// Console writes messages to stdout.
type Console struct {
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

// Silent stores messages internally, without writing them.
type Silent struct {
	msgs []opMsg
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
