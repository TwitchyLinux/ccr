package vts

import (
	"crypto/sha256"
	"errors"
	"fmt"
)

// Resource is a target representing a resource.
type Resource struct {
	Path string
	Name string
	Pos  *DefPosition

	Parent TargetRef
	Source *TargetRef

	Details []TargetRef
	Deps    []TargetRef
	Checks  []TargetRef

	Info RuntimeInfo
}

func (t *Resource) DefinedAt() *DefPosition {
	return t.Pos
}

func (t *Resource) IsClassTarget() bool {
	return false
}

func (t *Resource) TargetType() TargetType {
	return TargetResource
}

func (t *Resource) Class() TargetRef {
	return t.Parent
}

func (t *Resource) GlobalPath() string {
	return t.Path
}

func (t *Resource) TargetName() string {
	return t.Name
}

func (t *Resource) Dependencies() []TargetRef {
	return t.Deps
}

func (t *Resource) Checkers() []TargetRef {
	return t.Checks
}

func (t *Resource) Attributes() []TargetRef {
	return t.Details
}

func (t *Resource) Src() *TargetRef {
	return t.Source
}

func (t *Resource) RuntimeInfo() *RuntimeInfo {
	return &t.Info
}

func (t *Resource) Validate() error {
	if t.Parent.Target != nil {
		if _, ok := t.Parent.Target.(*ResourceClass); !ok {
			return fmt.Errorf("parent is type %T, but must be resource_class", t.Parent.Target)
		}
	} else if t.Parent.Path == "" {
		return errors.New("no parent attr_class specified")
	}
	if len(t.Parent.Constraints) > 0 {
		return errors.New("cannot specify constraints on a parent target")
	}

	if err := validateDetails(t.Details); err != nil {
		return err
	}
	if err := validateDeps(t.Deps, false); err != nil {
		return err
	}
	if t.Source != nil {
		if err := validateSource(*t.Source, t); err != nil {
			return err
		}
	}
	return nil
}

func (t *Resource) RollupHash(env *RunnerEnv, eval computeEval) ([]byte, error) {
	hash := sha256.New()
	fmt.Fprintf(hash, "%q\n%q\n%q\n", t.Path, t.Name, t.Parent.Target.(*ResourceClass).GlobalPath())

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
	if t.Source != nil {
		s, isHashable := t.Source.Target.(ReproducibleTarget)
		if !isHashable {
			return nil, WrapWithTarget(fmt.Errorf("cannot compute rollup hash on non-reproducible source target of type %T", t.Source.Target), t.Source.Target)
		}
		h, err := s.RollupHash(env, eval)
		if err != nil {
			return nil, err
		}
		hash.Write(h)
	}

	return hash.Sum(nil), nil
}
