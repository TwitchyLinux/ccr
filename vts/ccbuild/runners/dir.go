package runners

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"os"

	"github.com/twitchylinux/ccr/vts"
	"github.com/twitchylinux/ccr/vts/ccbuild/info"
	"go.starlark.net/starlark"
)

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

func (*dirCheckPresent) Run(r *vts.Resource, chkr *vts.Checker, opts *vts.RunnerEnv) error {
	d, err := r.RuntimeInfo().Get(info.StatPopulator, info.FileStat)
	if err != nil {
		return err
	}
	stat := d.(info.FileInfo)
	if !stat.IsDir() {
		return vts.WrapWithPath(fmt.Errorf("resource %q is not a directory", stat.Path), stat.Path)
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

func (*dirCheckPresent) PopulatorsNeeded() []vts.InfoPopulator {
	return []vts.InfoPopulator{info.StatPopulator}
}

// GenerateDir returns a generator runner that generates directories.
func GenerateDir() *dirGenerator {
	return &dirGenerator{}
}

type dirGenerator struct{}

func (*dirGenerator) String() string { return "dir.generator" }

func (*dirGenerator) Freeze() {}

func (*dirGenerator) Truth() starlark.Bool { return true }

func (*dirGenerator) Type() string { return "runner" }

func (t *dirGenerator) Hash() (uint32, error) {
	h := sha256.Sum256([]byte(fmt.Sprintf("%p", t)))
	return uint32(uint32(h[0]) + uint32(h[1])<<8 + uint32(h[2])<<16 + uint32(h[3])<<24), nil
}

func (*dirGenerator) Run(g *vts.Generator, inputs *vts.InputSet, opts *vts.RunnerEnv) error {
	p, err := resourcePath(inputs.Resource)
	if err != nil {
		return err
	}
	m, err := resourceMode(inputs.Resource)
	if err != nil {
		if err == errNoAttr {
			return errors.New("cannot generate dir when no mode was specified")
		}
		return err
	}

	return opts.FS.MkdirAll(p, m)
}
