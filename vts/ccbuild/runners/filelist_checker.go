package runners

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"os"

	"github.com/twitchylinux/ccr/confs/simple"
	"github.com/twitchylinux/ccr/vts"
	"go.starlark.net/starlark"
)

// FilelistCheckAllFiles returns a runner that checks a file containing a
// list of files conforms to all the specs mentioned.
func FilelistCheckAllFiles(wantMode os.FileMode, dirsOnly bool) *filelistChecker {
	return &filelistChecker{wantMode, dirsOnly}
}

type filelistChecker struct {
	wantMode os.FileMode
	DirsOnly bool
}

func (*filelistChecker) Kind() vts.CheckerKind { return vts.ChkKindEachResource }

func (*filelistChecker) String() string { return "filelist.checker" }

func (*filelistChecker) Freeze() {}

func (*filelistChecker) Truth() starlark.Bool { return true }

func (*filelistChecker) Type() string { return "runner" }

func (t *filelistChecker) Hash() (uint32, error) {
	h := sha256.Sum256([]byte(fmt.Sprintf("%p", t)))
	return uint32(uint32(h[0]) + uint32(h[1])<<8 + uint32(h[2])<<16 + uint32(h[3])<<24), nil
}

func (flc *filelistChecker) Run(r *vts.Resource, chkr *vts.Checker, opts *vts.RunnerEnv) error {
	path, err := resourcePath(r)
	if err != nil {
		if err == errNoAttr {
			return errors.New("no path specified")
		}
		return err
	}
	f, err := opts.FS.Open(path)
	if err != nil {
		return vts.WrapWithPath(err, path)
	}
	defer f.Close()
	fl, err := simple.Parse(simple.Config{Mode: simple.ModeLines}, f)
	if err != nil {
		return vts.WrapWithPath(err, path)
	}

	for i, line := range fl.Lines {
		s, err := opts.FS.Stat(line)
		if err != nil {
			return vts.WrapWithPath(fmt.Errorf("referencing file at line %d: %v", i+1, err), path)
		}

		if flc.wantMode > 0 && s.Mode()&os.ModePerm != flc.wantMode {
			return vts.WrapWithPath(fmt.Errorf("referenced file %q has mode %#o, want %#o", line, s.Mode()&os.ModePerm, flc.wantMode), path)
		}
		if flc.DirsOnly && s.IsDir() {
			return vts.WrapWithPath(fmt.Errorf("referenced file %q is directory", line), path)
		}
	}

	return nil
}

func (*filelistChecker) PopulatorsNeeded() []vts.InfoPopulator {
	return nil
}
