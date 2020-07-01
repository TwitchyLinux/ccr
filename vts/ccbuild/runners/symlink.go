package runners

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"os"

	"github.com/twitchylinux/ccr/vts"
	"go.starlark.net/starlark"
)

// SymlinkCheckPresent returns a runner that checks a symlink is present
// at the indicated path. If a target attribute is set, then it checks
// that the target matches.
func SymlinkCheckPresent() *symlinkCheckPresent {
	return &symlinkCheckPresent{}
}

type symlinkCheckPresent struct{}

func (*symlinkCheckPresent) Kind() vts.CheckerKind { return vts.ChkKindEachResource }

func (*symlinkCheckPresent) String() string { return "symlink.present" }

func (*symlinkCheckPresent) Freeze() {}

func (*symlinkCheckPresent) Truth() starlark.Bool { return true }

func (*symlinkCheckPresent) Type() string { return "runner" }

func (t *symlinkCheckPresent) Hash() (uint32, error) {
	h := sha256.Sum256([]byte(fmt.Sprintf("%p", t)))
	return uint32(uint32(h[0]) + uint32(h[1])<<8 + uint32(h[2])<<16 + uint32(h[3])<<24), nil
}

func (*symlinkCheckPresent) Run(r *vts.Resource, chkr *vts.Checker, opts *vts.RunnerEnv) error {
	path, err := resourcePath(r)
	if err != nil {
		if err == errNoAttr {
			return errors.New("no path specified")
		}
		return err
	}
	stat, err := opts.FS.Lstat(path)
	if err != nil {
		return vts.WrapWithPath(err, path)
	}
	if stat.IsDir() {
		return vts.WrapWithPath(fmt.Errorf("resource %q is a directory", path), path)
	}
	if stat.Mode()&os.ModeSymlink == 0 {
		return vts.WrapWithPath(fmt.Errorf("resource %q is not a symlink", path), path)
	}

	target, err := resourceTarget(r)
	if err != nil {
		if err == errNoAttr {
			return nil
		}
		return err
	}

	link, err := opts.FS.Readlink(path)
	if err != nil {
		return vts.WrapWithPath(err, path)
	}
	if link != target {
		return fmt.Errorf("symlink exists but its target %q does not match that of its resource", link)
	}
	return nil
}

func (*symlinkCheckPresent) PopulatorsNeeded() []vts.InfoPopulator {
	return nil
}

// GenerateSymlink returns a generator runner that generates symlinks.
func GenerateSymlink() *symlinkGenerator {
	return &symlinkGenerator{}
}

type symlinkGenerator struct{}

func (*symlinkGenerator) String() string { return "symlink.generator" }

func (*symlinkGenerator) Freeze() {}

func (*symlinkGenerator) Truth() starlark.Bool { return true }

func (*symlinkGenerator) Type() string { return "runner" }

func (t *symlinkGenerator) Hash() (uint32, error) {
	h := sha256.Sum256([]byte(fmt.Sprintf("%p", t)))
	return uint32(uint32(h[0]) + uint32(h[1])<<8 + uint32(h[2])<<16 + uint32(h[3])<<24), nil
}

func (*symlinkGenerator) Run(g *vts.Generator, inputs *vts.InputSet, opts *vts.RunnerEnv) error {
	p, err := resourcePath(inputs.Resource)
	if err != nil {
		return err
	}
	target, err := resourceTarget(inputs.Resource)
	if err != nil {
		if err == errNoAttr {
			return errors.New("cannot generate symlink when no target was specified")
		}
		return err
	}

	return opts.FS.Symlink(target, p)
}
