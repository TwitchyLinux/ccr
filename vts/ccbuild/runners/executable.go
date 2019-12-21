package runners

import (
	"crypto/sha256"
	"errors"
	"fmt"

	"github.com/twitchylinux/ccr/vts"
	"go.starlark.net/starlark"
)

// BinutilCheckComponent returns a runner that runs basic sanity checks over
// a component representing a cli binary.
func BinutilCheckComponent() *binutilCheckRunner {
	return &binutilCheckRunner{}
}

type binutilCheckRunner struct{}

func (*binutilCheckRunner) Kind() vts.CheckerKind { return vts.ChkKindEachComponent }

func (*binutilCheckRunner) String() string { return "binutil.sanity_check" }

func (*binutilCheckRunner) Freeze() {}

func (*binutilCheckRunner) Truth() starlark.Bool { return true }

func (*binutilCheckRunner) Type() string { return "runner" }

func (t *binutilCheckRunner) Hash() (uint32, error) {
	h := sha256.Sum256([]byte(fmt.Sprintf("%p", t)))
	return uint32(uint32(h[0]) + uint32(h[1])<<8 + uint32(h[2])<<16 + uint32(h[3])<<24), nil
}

func (r *binutilCheckRunner) Run(c *vts.Component, opts *vts.RunnerEnv) error {
	return errors.New("not implemented: " + r.String())
}
