package buildstep

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/twitchylinux/ccr/vts"
)

// RunShellCmd runs a shell command in the build environment.
func RunShellCmd(rb RunningBuild, step *vts.BuildStep) error {
	_, err := rb.ExecBlocking("/tmp", append([]string{"/bin/bash", "-c"}, step.Args...), os.Stdout, os.Stderr)
	return err
}

// RunPatch runs a patch command in the build environment.
func RunPatch(rb RunningBuild, step *vts.BuildStep) error {
	f, err := rb.SourceFS().Open(step.Path)
	if err != nil {
		return fmt.Errorf("reading patchfile: %v", err)
	}
	defer f.Close()

	dir := filepath.Join(rb.OverlayUpperPath(), step.ToPath)
	cmd := exec.Command("patch", fmt.Sprintf("-Np%d", step.PatchLevel))
	cmd.Dir = dir
	cmd.Stdin = f
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	return cmd.Run()
}
