package ccr

import (
	"fmt"

	"github.com/twitchylinux/ccr/vts"
)

type msgCategory string

// Common message categories.
const (
	MsgBadFind     msgCategory = "not found"
	MsgBadRef      msgCategory = "invalid reference"
	MsgFailedCheck msgCategory = "check failed"
)

type opTrack interface {
	Error(t vts.Target, category msgCategory, err error) error
	Warning(t vts.Target, category msgCategory, message string)
	Info(t vts.Target, category msgCategory, message string)
	IsInteractive() bool
}

type opMsg struct {
	isErr    bool
	category msgCategory
	msg      string
	err      error
	t        vts.Target
}

func printErr(target vts.Target, msg string, err error) {
	fmt.Printf("\033[1;31mError: \033[0m(%s) ", msg)
	if target != nil {
		if gt, ok := target.(vts.GlobalTarget); ok && gt.GlobalPath() != "" {
			fmt.Printf("\033[1;33m%s\033[0m: ", gt.GlobalPath())
		}
	}
	fmt.Printf("%v\n", err)

	if target != nil {
		if pos := target.DefinedAt(); pos != nil {
			fmt.Printf("  Failing target at \033[1;33m%s:%d:%d\033[0m\n", pos.Path, pos.Frame.Pos.Line, pos.Frame.Pos.Col)
		}
	}
	fmt.Println()
}

type consoleOpTrack struct {
}

func (t *consoleOpTrack) Error(target vts.Target, category msgCategory, err error) error {
	printErr(target, string(category), err)
	return err
}
func (t *consoleOpTrack) Warning(target vts.Target, category msgCategory, message string) {

}
func (t *consoleOpTrack) Info(target vts.Target, category msgCategory, message string) {

}
func (t *consoleOpTrack) IsInteractive() bool {
	return true
}

type silentOpTrack struct {
	msgs []opMsg
}

func (t *silentOpTrack) Error(target vts.Target, category msgCategory, err error) error {
	t.msgs = append(t.msgs, opMsg{
		t:        target,
		isErr:    true,
		category: category,
		err:      err,
	})
	return err
}
func (t *silentOpTrack) Warning(target vts.Target, category msgCategory, message string) {
	t.msgs = append(t.msgs, opMsg{
		t:        target,
		category: category,
		msg:      message,
	})
}
func (t *silentOpTrack) Info(target vts.Target, category msgCategory, message string) {
	// Info messages are not recorded for the silent op tracker.
}

func (t *silentOpTrack) IsInteractive() bool {
	return false
}
