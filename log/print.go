package log

import (
	"fmt"

	"github.com/twitchylinux/ccr/vts"
)

func printComputedValueErr(cv *vts.ComputedValue) {
	if len(cv.InlineScript) > 0 {
		fmt.Printf("   Originating from:  \033[1;33m%s\033[0m\n", "compute(<inline function>)")
	} else {
		fmt.Printf("   Originating from:  \033[1;33m%s\033[0m\n", fmt.Sprintf("compute(%q, %q)", cv.Filename, cv.Func))
	}

	fmt.Printf("         defined at:  \033[1;33m%s:%d:%d\033[0m\n", cv.Pos.Path, cv.Pos.Frame.Pos.Line, cv.Pos.Frame.Pos.Col)
	fmt.Println()
}

func printErrSource(kind MsgCategory, we vts.WrappedErr) {
	msg, thing := "Failing", "target"
	switch kind {
	case MsgBadDef:
		msg, thing = "Invalid", "target defined"
	default:
		if _, isConstErr := we.Err.(vts.FailingConstraintInfo); isConstErr {
			msg = "Constrained"
		}
	}

	switch {
	case we.Pos != nil:
		pos := we.Pos
		fmt.Printf("  %s %s at:  \033[1;33m%s:%d:%d\033[0m\n", msg, thing, pos.Path, pos.Frame.Pos.Line, pos.Frame.Pos.Col)
	case we.Target != nil:
		if pos := we.Target.DefinedAt(); pos != nil {
			fmt.Printf("  %s %s at:  \033[1;33m%s:%d:%d\033[0m\n", msg, thing, pos.Path, pos.Frame.Pos.Line, pos.Frame.Pos.Col)
		}
	}
	if we.ActionTarget != nil {
		if gt, ok := we.ActionTarget.(vts.GlobalTarget); ok {
			msg := "Failed by"
			if _, isConstErr := we.Err.(vts.FailingConstraintInfo); isConstErr {
				msg = "Constraint set on"
			}
			fmt.Printf("  %s %s:\n    \033[1;35m%s\033[0m\n", msg, gt.TargetType(), gt.GlobalPath())
		}
		if pos := we.ActionTarget.DefinedAt(); pos != nil {
			fmt.Printf("    Defined at \033[1;35m%s:%d:%d\033[0m\n", pos.Path, pos.Frame.Pos.Line, pos.Frame.Pos.Col)
		}
	}
}

func printErrBanner(msg MsgCategory, we vts.WrappedErr) {
	cta := "Error"
	if we.IsHostCheck {
		cta = "Host Error"
	}

	fmt.Printf("\033[1;31m%s: \033[0m(%s) ", cta, msg)
	if we.Target != nil {
		if gt, ok := we.Target.(vts.GlobalTarget); ok && gt.GlobalPath() != "" {
			fmt.Printf("\033[1;33m%s\033[0m: ", gt.GlobalPath())
		}
	}
	fmt.Printf("%v\n", we)
}

func printErr(msg MsgCategory, err error) {
	we, _ := err.(vts.WrappedErr)
	printErrBanner(msg, we)
	if c, isConstraint := we.Err.(vts.FailingConstraintInfo); isConstraint {
		fmt.Printf("  Failing constraint:     \033[1;33m%s\033[0m  %s  \033[1;33m%s\033[0m\n", c.Lhs, c.Op, c.Rhs)
	}

	if cv := we.ComputedValue; cv != nil {
		printComputedValueErr(cv)
	}

	if we.Path != "" {
		fmt.Printf("  Affected path at:   \033[1;33m%s\033[0m\n", we.Path)
	}

	printErrSource(msg, we)

	if len(we.TargetChain) > 0 {
		fmt.Println()
	}
	for _, target := range we.TargetChain {
		if pos := target.DefinedAt(); pos != nil {
			fmt.Printf("  Parent %s target at \033[1;33m%s:%d:%d\033[0m\n", target.TargetType(), pos.Path, pos.Frame.Pos.Line, pos.Frame.Pos.Col)
		}
	}

	fmt.Println()
}
