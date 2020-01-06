package runners

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"strings"

	"github.com/twitchylinux/ccr/vts"
	"go.starlark.net/starlark"
)

// CheckerOptCheckValid returns a runner that can check attrs
// are valid checker opt strings.
func CheckerOptCheckValid() *checkerOptValidRunner {
	return &checkerOptValidRunner{}
}

type checkerOptValidRunner struct{}

func (*checkerOptValidRunner) Kind() vts.CheckerKind { return vts.ChkKindEachAttr }

func (*checkerOptValidRunner) String() string { return "attr.checker_opt_valid" }

func (*checkerOptValidRunner) Freeze() {}

func (*checkerOptValidRunner) Truth() starlark.Bool { return true }

func (*checkerOptValidRunner) Type() string { return "runner" }

func (t *checkerOptValidRunner) Hash() (uint32, error) {
	h := sha256.Sum256([]byte(fmt.Sprintf("%p", t)))
	return uint32(uint32(h[0]) + uint32(h[1])<<8 + uint32(h[2])<<16 + uint32(h[3])<<24), nil
}

func (*checkerOptValidRunner) Run(attr *vts.Attr, chkr *vts.Checker, opts *vts.RunnerEnv) error {
	v, ok := attr.Value.(starlark.String)
	if !ok {
		return fmt.Errorf("expected string, got %T", attr.Value)
	}
	s := string(v)
	sepIdx := strings.Index(s, "=")
	if sepIdx < 0 {
		return errors.New("missing equals sign to delimit key value pair")
	}
	// TODO: Check individual key-value pairs.
	return nil
}
