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
	Error(category msgCategory, err error) error
	Warning(category msgCategory, message string)
	Info(category msgCategory, message string)
	IsInteractive() bool
}

type opMsg struct {
	isErr    bool
	category msgCategory
	msg      string
	err      error
	t        vts.Target
}

func printErr(msg string, err error) {
	we, _ := err.(vts.WrappedErr)
	fmt.Printf("\033[1;31mError: \033[0m(%s) ", msg)
	if we.Target != nil {
		if gt, ok := we.Target.(vts.GlobalTarget); ok && gt.GlobalPath() != "" {
			fmt.Printf("\033[1;33m%s\033[0m: ", gt.GlobalPath())
		}
	}
	fmt.Printf("%v\n", err)

	if we.Path != "" {
		fmt.Printf("  Artifact at \033[1;33m%s\033[0m\n", we.Path)
	}

	switch {
	case we.Pos != nil:
		pos := we.Pos
		fmt.Printf("  Failing target at \033[1;33m%s:%d:%d\033[0m\n", pos.Path, pos.Frame.Pos.Line, pos.Frame.Pos.Col)
	case we.Target != nil:
		if pos := we.Target.DefinedAt(); pos != nil {
			fmt.Printf("  Failing target at \033[1;33m%s:%d:%d\033[0m\n", pos.Path, pos.Frame.Pos.Line, pos.Frame.Pos.Col)
		}
	}
	if we.ActionTarget != nil {
		if gt, ok := we.ActionTarget.(vts.GlobalTarget); ok {
			fmt.Printf("  Failed by %s target: \033[1;33m%s\033[0m\n", gt.TargetType(), gt.GlobalPath())
		}
		if pos := we.ActionTarget.DefinedAt(); pos != nil {
			fmt.Printf("    Defined at \033[1;33m%s:%d:%d\033[0m\n", pos.Path, pos.Frame.Pos.Line, pos.Frame.Pos.Col)
		}
	}

	for _, target := range we.TargetChain {
		if pos := target.DefinedAt(); pos != nil {
			fmt.Printf("  Parent %s target at \033[1;33m%s:%d:%d\033[0m\n", target.TargetType(), pos.Path, pos.Frame.Pos.Line, pos.Frame.Pos.Col)
		}
	}

	fmt.Println()
}

type consoleOpTrack struct {
}

func (t *consoleOpTrack) Error(category msgCategory, err error) error {
	printErr(string(category), err)
	return err
}
func (t *consoleOpTrack) Warning(category msgCategory, message string) {

}
func (t *consoleOpTrack) Info(category msgCategory, message string) {

}
func (t *consoleOpTrack) IsInteractive() bool {
	return true
}

type silentOpTrack struct {
	msgs []opMsg
}

func (t *silentOpTrack) Error(category msgCategory, err error) error {
	t.msgs = append(t.msgs, opMsg{
		isErr:    true,
		category: category,
		err:      err,
	})
	return err
}
func (t *silentOpTrack) Warning(category msgCategory, message string) {
	t.msgs = append(t.msgs, opMsg{
		category: category,
		msg:      message,
	})
}
func (t *silentOpTrack) Info(category msgCategory, message string) {
	// Info messages are not recorded for the silent op tracker.
}

func (t *silentOpTrack) IsInteractive() bool {
	return false
}
