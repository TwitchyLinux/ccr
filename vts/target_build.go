package vts

import (
	"crypto/sha256"
	"fmt"

	"go.starlark.net/starlark"
)

// Build is a target representing a build.
type Build struct {
	Path string
	Name string
	Pos  *DefPosition
	Dir  string

	HostDeps []TargetRef
	Steps    []*BuildStep
}

func (t *Build) DefinedAt() *DefPosition {
	return t.Pos
}

func (t *Build) IsClassTarget() bool {
	return false
}

func (t *Build) TargetType() TargetType {
	return TargetBuild
}

func (t *Build) GlobalPath() string {
	return t.Path
}

func (t *Build) TargetName() string {
	return t.Name
}

func (t *Build) Validate() error {
	return nil
}

func (t *Build) Dependencies() []TargetRef {
	return t.HostDeps
}

func (t *Build) String() string {
	if t.Name == "" {
		return "build<$anonymous$>"
	}
	return fmt.Sprintf("build<%s>", t.Name)
}

func (t *Build) Freeze() {

}

func (t *Build) Truth() starlark.Bool {
	return true
}

func (t *Build) Type() string {
	return "build"
}

func (t *Build) Hash() (uint32, error) {
	h := sha256.Sum256([]byte(fmt.Sprintf("%p", t)))
	return uint32(uint32(h[0]) + uint32(h[1])<<8 + uint32(h[2])<<16 + uint32(h[3])<<24), nil
}

func (t *Build) RollupHash(env *RunnerEnv, eval computeEval) ([]byte, error) {
	hash := sha256.New()
	fmt.Fprintf(hash, "build: %q\n%q\n", t.Path, t.Name)

	for _, dep := range t.HostDeps {
		rt, isHashable := dep.Target.(ReproducibleTarget)
		if !isHashable {
			return nil, WrapWithTarget(fmt.Errorf("cannot compute rollup hash on non-reproducible target of type %T", dep.Target), dep.Target)
		}
		h, err := rt.RollupHash(env, eval)
		if err != nil {
			return nil, err
		}
		hash.Write(h)
	}
	for _, step := range t.Steps {
		h, err := step.RollupHash(env, eval)
		if err != nil {
			return nil, err
		}
		hash.Write(h)
	}

	return hash.Sum(nil), nil
}
