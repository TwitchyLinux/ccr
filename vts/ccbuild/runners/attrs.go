package runners

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"strings"

	"github.com/twitchylinux/ccr/vts"
	"go.starlark.net/starlark"
)

// PathCheckValid returns a runner that can check attrs
// are valid paths.
func PathCheckValid() *pathValidRunner {
	return &pathValidRunner{}
}

type pathValidRunner struct{}

func (*pathValidRunner) Kind() vts.CheckerKind { return vts.ChkKindEachAttr }

func (*pathValidRunner) String() string { return "attr.path_valid" }

func (*pathValidRunner) Freeze() {}

func (*pathValidRunner) Truth() starlark.Bool { return true }

func (*pathValidRunner) Type() string { return "runner" }

func (t *pathValidRunner) Hash() (uint32, error) {
	h := sha256.Sum256([]byte(fmt.Sprintf("%p", t)))
	return uint32(uint32(h[0]) + uint32(h[1])<<8 + uint32(h[2])<<16 + uint32(h[3])<<24), nil
}

func (*pathValidRunner) Run(attr *vts.Attr, opts *vts.CheckerOpts) error {
	path := attr.Value.String()
	if path == "" {
		return errors.New("empty path")
	}
	if strings.ContainsAny(path, "\x00:<>") {
		return errors.New("path contains illegal characters")
	}
	return nil
}
