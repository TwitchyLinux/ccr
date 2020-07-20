package vts

import (
	"crypto/sha256"
	"fmt"
)

// Component is a target representing a component.
type Component struct {
	Path string
	Name string
	Pos  *DefPosition

	Details []TargetRef
	Deps    []TargetRef
	Checks  []TargetRef

	Info RuntimeInfo
}

func (t *Component) DefinedAt() *DefPosition {
	return t.Pos
}

func (t *Component) IsClassTarget() bool {
	return false
}

func (t *Component) TargetType() TargetType {
	return TargetComponent
}

func (t *Component) GlobalPath() string {
	return t.Path
}

func (t *Component) TargetName() string {
	return t.Name
}

func (t *Component) Dependencies() []TargetRef {
	return t.Deps
}

func (t *Component) Checkers() []TargetRef {
	return t.Checks
}

func (t *Component) Attributes() []TargetRef {
	return t.Details
}

func (t *Component) RuntimeInfo() *RuntimeInfo {
	return &t.Info
}

func (t *Component) Validate() error {
	if err := validateDetails(t.Details); err != nil {
		return err
	}
	if err := validateDeps(t.Deps); err != nil {
		return err
	}
	return nil
}

func (t *Component) RollupHash(env *RunnerEnv, eval computeEval) ([]byte, error) {
	hash := sha256.New()
	fmt.Fprintf(hash, "%q\n%q\n", t.Path, t.Name)

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
