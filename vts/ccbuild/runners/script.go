package runners

import (
	"bufio"
	"crypto/sha256"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/twitchylinux/ccr/vts"
	"github.com/twitchylinux/ccr/vts/ccbuild/info"
	"go.starlark.net/starlark"
)

// ScriptCheckValid returns a runner that can check resources
// reference a well-formed script file.
func ScriptCheckValid() *scriptValidRunner {
	return &scriptValidRunner{}
}

type scriptValidRunner struct{}

func (*scriptValidRunner) Kind() vts.CheckerKind { return vts.ChkKindEachResource }

func (*scriptValidRunner) String() string { return "script.check_valid" }

func (*scriptValidRunner) Freeze() {}

func (*scriptValidRunner) Truth() starlark.Bool { return true }

func (*scriptValidRunner) Type() string { return "runner" }

func (t *scriptValidRunner) Hash() (uint32, error) {
	h := sha256.Sum256([]byte(fmt.Sprintf("%p", t)))
	return uint32(uint32(h[0]) + uint32(h[1])<<8 + uint32(h[2])<<16 + uint32(h[3])<<24), nil
}

func (*scriptValidRunner) Run(resource *vts.Resource, chkr *vts.Checker, opts *vts.RunnerEnv) error {
	d, err := resource.RuntimeInfo().Get(info.StatPopulator, info.FileStat)
	if err != nil {
		return err
	}
	fileInfo := d.(info.FileInfo)

	if fileInfo.Mode()&os.ModePerm&0111 == 0 {
		return vts.WrapWithPath(fmt.Errorf("script is not executable: %#o", fileInfo.Mode()), fileInfo.Path)
	}

	f, err := opts.FS.Open(fileInfo.Path)
	if err != nil {
		return vts.WrapWithPath(err, fileInfo.Path)
	}
	defer f.Close()
	firstLine, err := bufio.NewReader(f).ReadString('\n')
	if err != nil {
		return vts.WrapWithPath(fmt.Errorf("reading script header: %v", err), fileInfo.Path)
	}
	if !strings.HasPrefix(firstLine, "#!") {
		return vts.WrapWithPath(errors.New("sheban missing from script"), fileInfo.Path)
	}
	interp := strings.Split(strings.TrimSpace(strings.TrimPrefix(firstLine, "#!")), " ")[0]

	t, err := opts.Universe.FindByPath(interp, opts)
	if err != nil {
		return vts.WrapWithPath(fmt.Errorf("script interpreter %q not present", interp), interp)
	}
	interpR, ok := t.(*vts.Resource)
	if !ok {
		return vts.WrapWithPath(fmt.Errorf("target representing script interpreter is %v, not a resource", t.TargetType()), interp)
	}
	switch c := interpR.Parent.Target.(*vts.ResourceClass).GlobalPath(); c {
	case "common://resources:binary", "common://resources:binary_symlink", "common://resources:script":
		// Sweet, it exists, and will be validated as an executable.
	default:
		return vts.WrapWithPath(fmt.Errorf("script intepreter is of non-executable class %s", c), interp)
	}

	return nil
}

func (*scriptValidRunner) PopulatorsNeeded() []vts.InfoPopulator {
	return []vts.InfoPopulator{info.StatPopulator}
}
