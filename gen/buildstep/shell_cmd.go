package buildstep

import (
	"io"

	"github.com/twitchylinux/ccr/vts"
)

// RunShellCmd runs a shell command in the build environment.
func RunShellCmd(rb RunningBuild, step *vts.BuildStep, o, e io.Writer) error {
	_, err := rb.ExecBlocking("/tmp", append([]string{"/bin/bash", "-c"}, step.Args...), o, e)
	return err
}
