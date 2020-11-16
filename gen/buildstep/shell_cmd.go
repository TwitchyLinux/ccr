package buildstep

import (
	"io"

	"github.com/twitchylinux/ccr/vts"
)

// RunShellCmd runs a shell command in the build environment.
func RunShellCmd(rb RunningBuild, step *vts.BuildStep, o, e io.Writer) error {
	if len(step.Args) > 0 {
		step.Args[0] = "set +h;umask 022;" + step.Args[0]
	}
	_, err := rb.ExecBlocking("/tmp", append([]string{"/bin/bash", "-c"}, step.Args...), o, e)
	return err
}
