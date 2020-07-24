package buildstep

import (
	"fmt"
	"os"

	"github.com/twitchylinux/ccr/vts"
)

// RunShellCmd runs a shell command in the build environment.
func RunShellCmd(rb RunningBuild, step *vts.BuildStep) error {
	if err := rb.ExecBlocking(append([]string{"/bin/bash", "-c"}, step.Args...), os.Stdout, os.Stderr); err != nil {
		return fmt.Errorf("shell execution failed: %v", err)
	}
	return nil
}
