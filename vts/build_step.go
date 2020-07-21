package vts

import (
	"crypto/sha256"
	"fmt"

	"go.starlark.net/starlark"
)

type StepKind string

// Valid BuildStep StepKind values.
const (
	StepUnpackGz = "unpack_gz"
)

// BuildStep is an anonymous target representing a step in a build.
type BuildStep struct {
	Pos  *DefPosition
	Kind StepKind
	Dir  string

	ToPath string
	Path   string
	URL    string
	SHA256 string
}

func (t *BuildStep) DefinedAt() *DefPosition {
	return t.Pos
}

func (t *BuildStep) IsClassTarget() bool {
	return false
}

func (t *BuildStep) Validate() error {
	return nil
}

func (t *BuildStep) String() string {
	return fmt.Sprintf("build_step<%s>", t.Kind)
}

func (t *BuildStep) Freeze() {

}

func (t *BuildStep) Truth() starlark.Bool {
	return true
}

func (t *BuildStep) Type() string {
	return "build_step"
}

func (t *BuildStep) Hash() (uint32, error) {
	h := sha256.Sum256([]byte(fmt.Sprintf("%p", t)))
	return uint32(uint32(h[0]) + uint32(h[1])<<8 + uint32(h[2])<<16 + uint32(h[3])<<24), nil
}

func (t *BuildStep) RollupHash(env *RunnerEnv, eval computeEval) ([]byte, error) {
	hash := sha256.New()
	fmt.Fprintf(hash, "step: %q\n%q\n", t.Kind, t.Dir)
	fmt.Fprintf(hash, "step: %q\n%q\n%q\n%q\n", t.ToPath, t.Path, t.URL, t.SHA256)

	return hash.Sum(nil), nil
}
