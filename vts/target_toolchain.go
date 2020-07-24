package vts

import (
	"crypto/sha256"
	"fmt"
	"sort"
)

// Toolchain is a target representing a specific host toolchain.
type Toolchain struct {
	Path string
	Name string
	Pos  *DefPosition

	Details        []TargetRef
	BinaryMappings map[string]string
	Deps           []TargetRef
}

func (t *Toolchain) DefinedAt() *DefPosition {
	return t.Pos
}

func (t *Toolchain) IsClassTarget() bool {
	return false
}

func (t *Toolchain) TargetType() TargetType {
	return TargetToolchain
}

func (t *Toolchain) GlobalPath() string {
	return t.Path
}

func (t *Toolchain) TargetName() string {
	return t.Name
}

func (t *Toolchain) Validate() error {
	if err := validateDetails(t.Details); err != nil {
		return err
	}
	if err := validateDeps(t.Deps, false); err != nil {
		return err
	}
	return nil
}

func (t *Toolchain) RollupHash(env *RunnerEnv, eval computeEval) ([]byte, error) {
	hash := sha256.New()
	fmt.Fprintf(hash, "%q\n%q\n", t.Name, t.Path)

	orderedMappings := make([]string, len(t.BinaryMappings))
	c := 0
	for name, path := range t.BinaryMappings {
		orderedMappings[c] = fmt.Sprintf("%s=%s\n", name, path)
		c++
	}
	sort.Strings(orderedMappings)
	for _, s := range orderedMappings {
		fmt.Fprint(hash, s)
	}

	for _, attr := range t.Details {
		a := attr.Target.(*Attr)
		fmt.Fprintf(hash, "%q\n%q\n%q\n", a.Name, a.Path, a.Parent.Target.(*AttrClass).GlobalPath())
		// TODO: Hash attribute class.
		if cv, isComputedValue := a.Val.(*ComputedValue); isComputedValue {
			fmt.Fprintf(hash, "computed params: file = %q func = %q inline = %q", cv.Filename, cv.Func, string(cv.InlineScript))
		}
		v, err := a.Value(t, env, eval)
		if err != nil {
			return nil, WrapWithTarget(err, a)
		}
		fmt.Fprint(hash, v)
	}
	for _, dep := range t.Deps {
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

	return hash.Sum(nil), nil
}

func (t *Toolchain) Dependencies() []TargetRef {
	return t.Deps
}
func (t *Toolchain) Attributes() []TargetRef {
	return t.Details
}
