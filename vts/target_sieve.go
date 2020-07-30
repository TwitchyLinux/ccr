package vts

import (
	"crypto/sha256"
	"fmt"

	"go.starlark.net/starlark"
)

// Sieve is a target used to filter and combine the output from generators.
type Sieve struct {
	Pos          *DefPosition
	Name         string
	TargetPath   string
	ContractPath string

	Inputs []TargetRef

	ExcludeGlobs []string
}

func (t *Sieve) DefinedAt() *DefPosition {
	return t.Pos
}

func (t *Sieve) IsClassTarget() bool {
	return false
}

func (t *Sieve) GlobalPath() string {
	return t.TargetPath
}

func (t *Sieve) TargetName() string {
	return t.Name
}

func (t *Sieve) TargetType() TargetType {
	return TargetSieve
}

func (t *Sieve) Validate() error {
	for i, inp := range t.Inputs {
		_, puesdo := inp.Target.(*Puesdo)
		_, build := inp.Target.(*Build)
		if !puesdo && !build {
			return fmt.Errorf("inputs[%d] is type %T, but must be build, file, or deb", i, inp.Target)
		}
		if len(inp.Constraints) > 0 {
			return fmt.Errorf("inputs[%d]: cannot specify constraints here", i)
		}
	}
	return nil
}

func (t *Sieve) NeedInputs() []TargetRef {
	return t.Inputs
}

func (t *Sieve) String() string {
	return fmt.Sprintf("sieve<%s>", "_")
}

func (t *Sieve) Freeze() {

}

func (t *Sieve) Truth() starlark.Bool {
	return true
}

func (t *Sieve) Type() string {
	return "sieve"
}

func (t *Sieve) Hash() (uint32, error) {
	h := sha256.Sum256([]byte(fmt.Sprintf("%p", t)))
	return uint32(uint32(h[0]) + uint32(h[1])<<8 + uint32(h[2])<<16 + uint32(h[3])<<24), nil
}

func (t *Sieve) RollupHash(env *RunnerEnv, eval computeEval) ([]byte, error) {
	hash := sha256.New()
	fmt.Fprintf(hash, "Sieve: %q\n", t.Name)

	for _, dep := range t.Inputs {
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
	for _, ex := range t.ExcludeGlobs {
		fmt.Fprintf(hash, "Ex pattern: %s\n", ex)
	}

	return hash.Sum(nil), nil
}
