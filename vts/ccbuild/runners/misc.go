package runners

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"os"

	"github.com/twitchylinux/ccr/vts"
	"go.starlark.net/starlark"
)

// NoopCheckComponent returns a runner that never returns an error.
func NoopCheckComponent() *noopCheckRunner {
	return &noopCheckRunner{}
}

type noopCheckRunner struct{}

func (*noopCheckRunner) Kind() vts.CheckerKind { return vts.ChkKindEachComponent }

func (*noopCheckRunner) String() string { return "misc.noop" }

func (*noopCheckRunner) Freeze() {}

func (*noopCheckRunner) Truth() starlark.Bool { return true }

func (*noopCheckRunner) Type() string { return "runner" }

func (t *noopCheckRunner) Hash() (uint32, error) {
	h := sha256.Sum256([]byte(fmt.Sprintf("%p", t)))
	return uint32(uint32(h[0]) + uint32(h[1])<<8 + uint32(h[2])<<16 + uint32(h[3])<<24), nil
}

func (*noopCheckRunner) Run(c *vts.Component, opts *vts.RunnerEnv) error {
	return nil
}

// FileCheckPresent returns a runner that checks a file is present
// at the indicated path.
func FileCheckPresent() *fileCheckPresent {
	return &fileCheckPresent{}
}

type fileCheckPresent struct{}

func (*fileCheckPresent) Kind() vts.CheckerKind { return vts.ChkKindEachResource }

func (*fileCheckPresent) String() string { return "file.present" }

func (*fileCheckPresent) Freeze() {}

func (*fileCheckPresent) Truth() starlark.Bool { return true }

func (*fileCheckPresent) Type() string { return "runner" }

func (t *fileCheckPresent) Hash() (uint32, error) {
	h := sha256.Sum256([]byte(fmt.Sprintf("%p", t)))
	return uint32(uint32(h[0]) + uint32(h[1])<<8 + uint32(h[2])<<16 + uint32(h[3])<<24), nil
}

func (*fileCheckPresent) Run(r *vts.Resource, opts *vts.RunnerEnv) error {
	path, err := resourcePath(r)
	if err != nil {
		if err == errNoAttr {
			return errors.New("no path specified")
		}
		return err
	}
	stat, err := opts.FS.Stat(path)
	if err != nil {
		return vts.WrapWithPath(err, path)
	}
	if stat.IsDir() {
		return vts.WrapWithPath(fmt.Errorf("resource %q is a directory", path), path)
	}
	return nil
}

// DirCheckPresent returns a runner that checks a directory is present
// at the indicated path.
func DirCheckPresent() *dirCheckPresent {
	return &dirCheckPresent{}
}

type dirCheckPresent struct{}

func (*dirCheckPresent) Kind() vts.CheckerKind { return vts.ChkKindEachResource }

func (*dirCheckPresent) String() string { return "dir.present" }

func (*dirCheckPresent) Freeze() {}

func (*dirCheckPresent) Truth() starlark.Bool { return true }

func (*dirCheckPresent) Type() string { return "runner" }

func (t *dirCheckPresent) Hash() (uint32, error) {
	h := sha256.Sum256([]byte(fmt.Sprintf("%p", t)))
	return uint32(uint32(h[0]) + uint32(h[1])<<8 + uint32(h[2])<<16 + uint32(h[3])<<24), nil
}

func (*dirCheckPresent) Run(r *vts.Resource, opts *vts.RunnerEnv) error {
	path, err := resourcePath(r)
	if err != nil {
		return err
	}
	stat, err := opts.FS.Stat(path)
	if err != nil {
		return vts.WrapWithPath(err, path)
	}
	if !stat.IsDir() {
		return vts.WrapWithPath(fmt.Errorf("resource %q is not a directory", path), path)
	}

	m, err := resourceMode(r)
	if err == errNoAttr {
		return nil
	}
	if err != nil {
		return err
	}
	if stat.Mode()&os.ModePerm != m&os.ModePerm {
		return fmt.Errorf("permissions mismatch: %#o was specified but directory is %#o", m, stat.Mode()&os.ModePerm)
	}
	return nil
}
