package runners

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/twitchylinux/ccr/vts"
	"github.com/twitchylinux/ccr/vts/ccbuild/info"
	"go.starlark.net/starlark"
	"gopkg.in/src-d/go-billy.v4"
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

	m, err := resourceMode(r, opts)
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

// DirOnlyCxxHeaders returns a runner that checks a directory contains
// only C headers and other directories.
func DirOnlyCxxHeaders() *cxxHeadersDirCheck {
	return &cxxHeadersDirCheck{}
}

type cxxHeadersDirCheck struct{}

func (*cxxHeadersDirCheck) Kind() vts.CheckerKind { return vts.ChkKindEachResource }

func (*cxxHeadersDirCheck) String() string { return "dir.c_headers" }

func (*cxxHeadersDirCheck) Freeze() {}

func (*cxxHeadersDirCheck) Truth() starlark.Bool { return true }

func (*cxxHeadersDirCheck) Type() string { return "runner" }

func (t *cxxHeadersDirCheck) Hash() (uint32, error) {
	h := sha256.Sum256([]byte(fmt.Sprintf("%p", t)))
	return uint32(uint32(h[0]) + uint32(h[1])<<8 + uint32(h[2])<<16 + uint32(h[3])<<24), nil
}

func (c *cxxHeadersDirCheck) checkDir(fs billy.Filesystem, path string) error {
	dirStat, err := fs.Stat(path)
	if err != nil {
		return vts.WrapWithPath(err, path)
	}
	if !dirStat.IsDir() {
		return vts.WrapWithPath(errors.New("expected directory, found file instead"), path)
	}
	if m := dirStat.Mode(); (m & 0555) == 0 {
		return vts.WrapWithPath(fmt.Errorf("unusable permissions on C header directory: %v", m), path)
	}

	files, err := fs.ReadDir(path)
	if err != nil {
		return vts.WrapWithPath(err, path)
	}
	for _, f := range files {
		if f.IsDir() {
			if err := c.checkDir(fs, fs.Join(path, f.Name())); err != nil {
				return err
			}
			continue
		}

		if m := f.Mode(); (m & 0111) != 0 {
			return vts.WrapWithPath(fmt.Errorf("unreadable C header: %v", m), fs.Join(path, f.Name()))
		}
		if n := f.Name(); !strings.HasSuffix(n, ".h") {
			return vts.WrapWithPath(fmt.Errorf("file is not a C header: extension is %q", filepath.Ext(f.Name())), fs.Join(path, f.Name()))
		}
	}
	return nil
}

func (c *cxxHeadersDirCheck) Run(r *vts.Resource, chkr *vts.Checker, opts *vts.RunnerEnv) error {
	d, err := r.RuntimeInfo().Get(info.StatPopulator, info.FileStat)
	if err != nil {
		return err
	}
	stat := d.(info.FileInfo)
	if !stat.IsDir() {
		return vts.WrapWithPath(fmt.Errorf("resource %q is not a directory", stat.Path), stat.Path)
	}

	return c.checkDir(opts.FS, stat.Path)
}

func (*cxxHeadersDirCheck) PopulatorsNeeded() []vts.InfoPopulator {
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
	p, err := resourcePath(inputs.Resource, opts)
	if err != nil {
		return err
	}
	m, err := resourceMode(inputs.Resource, opts)
	if err != nil {
		if err == errNoAttr {
			return errors.New("cannot generate dir when no mode was specified")
		}
		return err
	}

	return opts.FS.MkdirAll(p, m)
}
