package ccr

import (
	"fmt"

	"github.com/twitchylinux/ccr/vts"
)

type msgCategory string

// Common message categories.
const (
	MsgBadFind     msgCategory = "find failed"
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

func printCheckFailedErr(target vts.Target, err error) {
	fmt.Print("\033[1;31mError: \033[0m")
	if target != nil {
		if gt, ok := target.(vts.GlobalTarget); ok {
			if pos := target.DefinedAt(); pos != nil {
				fmt.Printf("\033[1;33m%s %d:%d\033[0m ", gt.GlobalPath(), pos.Frame.Pos.Line, pos.Frame.Pos.Col)
			} else {
				fmt.Printf("\033[1;33m%s\033[0m ", gt.GlobalPath())
			}
		}
	}
	fmt.Printf("Check failed: %v\n", err)
	fmt.Println()
}

type consoleOpTrack struct {
	msgs []opMsg
}

func (t *consoleOpTrack) Error(target vts.Target, category msgCategory, err error) error {
	switch category {
	case MsgFailedCheck:
		printCheckFailedErr(target, err)

	}
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
