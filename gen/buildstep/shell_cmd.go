package buildstep

import (
	"os"

	"github.com/twitchylinux/ccr/vts"
)

// RunShellCmd runs a shell command in the build environment.
func RunShellCmd(rb RunningBuild, step *vts.BuildStep) error {
	_, err := rb.ExecBlocking("/tmp", append([]string{"/bin/bash", "-c"}, step.Args...), os.Stdout, os.Stderr)
	return err
}
