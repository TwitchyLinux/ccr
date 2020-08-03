package buildstep

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/twitchylinux/ccr/vts"
)

// RunConfigure runs a configure script in the build environment.
func RunConfigure(rb RunningBuild, step *vts.BuildStep) error {
	cmd, wd := filepath.Join(step.Dir, "configure"), step.Dir
	if step.Path != "" {
		cmd = step.Path
	}

	args := make([]string, 0, len(step.NamedArgs))
	for k, v := range step.NamedArgs {
		args = append(args, fmt.Sprintf("--%s=%s", k, v))
	}
	args = append(args, "TMPDIR=/tmp")

	// fmt.Fprintf(os.Stdout, "+ Running %s %s from %s.\n", cmd, strings.Join(args, " "), wd)
	_, err := rb.ExecBlocking(wd, append([]string{cmd}, args...), os.Stdout, os.Stderr)
	if err != nil {
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
