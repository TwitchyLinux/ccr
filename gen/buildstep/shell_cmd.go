package buildstep

import (
	"fmt"
	"os"

	"github.com/twitchylinux/ccr/vts"
)

// RunShellCmd runs a shell command in the build environment.
func RunShellCmd(rb RunningBuild, step *vts.BuildStep) error {
	ec, err := rb.ExecBlocking(append([]string{"/bin/bash", "-c"}, step.Args...), os.Stdout, os.Stderr)
	if err != nil {
		return fmt.Errorf("shell execution failed: %v", err)
	}
	if ec != 0 {
		return fmt.Errorf("exit status %d", ec)
	}
	return nil
}
