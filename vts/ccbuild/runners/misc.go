package runners

import (
	"crypto/sha256"
	"fmt"

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
