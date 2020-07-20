package runners

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"strings"

	version "github.com/knqyf263/go-deb-version"
	"github.com/twitchylinux/ccr/proc"
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

func (*noopCheckRunner) Run(c *vts.Component, chkr *vts.Checker, opts *vts.RunnerEnv) error {
	return nil
}

func (*noopCheckRunner) PopulatorsNeeded() []vts.InfoPopulator {
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

func (*fileCheckPresent) Run(r *vts.Resource, chkr *vts.Checker, opts *vts.RunnerEnv) error {
	path, err := resourcePath(r, opts)
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

func (*fileCheckPresent) PopulatorsNeeded() []vts.InfoPopulator {
	return nil
}

// SemverCheckValid returns a runner that can check attrs
// are valid semantic version strings.
func SemverCheckValid() *semverValidRunner {
	return &semverValidRunner{}
}

type semverValidRunner struct{}

func (*semverValidRunner) Kind() vts.CheckerKind { return vts.ChkKindEachAttr }

func (*semverValidRunner) String() string { return "attr.checker_semver_valid" }

func (*semverValidRunner) Freeze() {}

func (*semverValidRunner) Truth() starlark.Bool { return true }

func (*semverValidRunner) Type() string { return "runner" }

func (t *semverValidRunner) Hash() (uint32, error) {
	h := sha256.Sum256([]byte(fmt.Sprintf("%p", t)))
	return uint32(uint32(h[0]) + uint32(h[1])<<8 + uint32(h[2])<<16 + uint32(h[3])<<24), nil
}

func (*semverValidRunner) Run(attr *vts.Attr, chkr *vts.Checker, opts *vts.RunnerEnv) error {
	sv, err := attr.Value(chkr, opts, proc.EvalComputedAttribute)
	if err != nil {
		return err
	}
	v, ok := sv.(starlark.String)
	if !ok {
		return fmt.Errorf("expected string, got %T", sv)
	}
	s := string(v)

	if strings.ContainsAny(s, " \n\r\t") {
		return fmt.Errorf("invalid version %q: semvers cannot contain whitespace", s)
	}
	if strings.Contains(s, ":") {
		return fmt.Errorf("invalid version %q: semvers cannot contain a trailing epoch", s)
	}

	if _, err := version.NewVersion(s); err != nil {
		return fmt.Errorf("invalid version %q: %v", s, err)
	}
	return nil
}
