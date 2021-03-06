package buildstep

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/twitchylinux/ccr/vts"
)

// RunConfigure runs a configure script in the build environment.
func RunConfigure(rb RunningBuild, step *vts.BuildStep, o, e io.Writer) error {
	cmd, wd := filepath.Join(step.Dir, "configure"), step.Dir
	if step.Path != "" {
		cmd = step.Path
	}

	args := make([]string, 0, len(step.NamedArgs))
	for k, v := range step.NamedArgs {
		if v != "" {
			args = append(args, fmt.Sprintf("--%s=%s", k, v))
		} else {
			args = append(args, fmt.Sprintf("--%s", k))
		}
	}
	for _, a := range step.Args {
		args = append(args, a)
	}

	// fmt.Fprintf(os.Stdout, "+ Running %s %s from %s.\n", cmd, strings.Join(args, " "), wd)
	ec, err := rb.ExecBlocking(wd, append([]string{cmd}, args...), o, e)
	if err != nil || ec != 0 {
		p := filepath.Join(rb.OverlayUpperPath(), step.Dir, "config.log")
		if _, err := os.Stat(p); err == nil {
			fmt.Fprintln(os.Stderr, "Configure exited with exit code!!\nA config.log file was found, copying to /tmp/ccr_config.log")
			if err := exec.Command("cp", p, "/tmp/ccr_config.log").Run(); err != nil {
				return fmt.Errorf("failed copying configure log: %v", err)
			}
		}
	}
	return err
}
