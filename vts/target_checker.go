package vts

import (
	"crypto/sha256"
	"fmt"

	"go.starlark.net/starlark"
)

type CheckerKind string

// Valid checker kinds.
const (
	ChkKindEachResource  CheckerKind = "each_resource"
	ChkKindEachAttr      CheckerKind = "each_attr"
	ChkKindEachComponent CheckerKind = "each_component"
	ChkKindGlobal        CheckerKind = "global"
)

// Checker is a target describing a check on a target or the system.
type Checker struct {
	Path string
	Name string
	Kind CheckerKind
	Pos  *DefPosition

	Runner starlark.Value
}

func (t *Checker) DefinedAt() *DefPosition {
	return t.Pos
}

func (t *Checker) IsClassTarget() bool {
	return false
}

func (t *Checker) TargetType() TargetType {
	return TargetChecker
}

func (t *Checker) GlobalPath() string {
	return t.Path
}

func (t *Checker) TargetName() string {
	return t.Name
}

func (t *Checker) Validate() error {
	switch t.Kind {
	case ChkKindEachResource:
		if _, ok := t.Runner.(eachResourceRunner); !ok {
			return fmt.Errorf("runner %v incompatible with kind = %q", t.Runner, t.Kind)
		}
	case ChkKindEachAttr:
		if _, ok := t.Runner.(eachAttrRunner); !ok {
			return fmt.Errorf("runner %v incompatible with kind = %q", t.Runner, t.Kind)
		}
	case ChkKindEachComponent:
		if _, ok := t.Runner.(eachComponentRunner); !ok {
			return fmt.Errorf("runner %v incompatible with kind = %q", t.Runner, t.Kind)
		}
	case ChkKindGlobal:
		if _, ok := t.Runner.(globalRunner); !ok {
			return fmt.Errorf("runner %v incompatible with kind = %q", t.Runner, t.Kind)
		}
	default:
		return fmt.Errorf("invalid checker kind: %q", t.Kind)
	}
	return nil
}

func (t *Checker) String() string {
	return fmt.Sprintf("checker<%s, %s>", t.Name, t.Kind)
}

func (t *Checker) Freeze() {

}

func (t *Checker) Truth() starlark.Bool {
	return true
}

func (t *Checker) Type() string {
	return "checker"
}

func (t *Checker) Hash() (uint32, error) {
	h := sha256.Sum256([]byte(fmt.Sprintf("%p", t)))
	return uint32(uint32(h[0]) + uint32(h[1])<<8 + uint32(h[2])<<16 + uint32(h[3])<<24), nil
}

// RunAttr runs the checker on an attribute. This method should be called
// through the attribute's class.
func (t *Checker) RunAttr(attr *Attr, opts *RunnerEnv) error {
	if t.Kind != ChkKindEachAttr {
		return fmt.Errorf("checker %v cannot be used to check a attr", t.Kind)
	}
	ac, ok := t.Runner.(eachAttrRunner)
	if !ok {
		return fmt.Errorf("runner %v cannot be used to check a attr", t.Runner)
	}
	err := ac.Run(attr, t, opts)
	if err != nil {
		err = WrapWithTarget(err, attr)
		err = WrapWithActionTarget(err, t)
	}
	return err
}

// RunResource runs the checker on a resource. This method should be called
// through the resource class.
func (t *Checker) RunResource(r *Resource, opts *RunnerEnv) error {
	if t.Kind != ChkKindEachResource {
		return fmt.Errorf("checker %v cannot be used to check a resource", t.Kind)
	}
	ac, ok := t.Runner.(eachResourceRunner)
	if !ok {
		return fmt.Errorf("runner %v cannot be used to check a resource", t.Runner)
	}

	for _, pop := range ac.PopulatorsNeeded() {
		ri := r.RuntimeInfo()
		if ri.HasRun(pop) {
			continue
		}
		if err := pop.Run(r, opts, ri); err != nil {
			err = WrapWithTarget(err, r)
			err = WrapWithActionTarget(err, t)
			return err
		}
	}

	err := ac.Run(r, t, opts)
	if err != nil {
		err = WrapWithTarget(err, r)
		err = WrapWithActionTarget(err, t)
	}
	return err
}

// RunCheckedTarget is invoked when running checks directly on the target
// on which they are defined.
func (t *Checker) RunCheckedTarget(tgt CheckedTarget, opts *RunnerEnv) error {
	switch t.Kind {
	case ChkKindGlobal:
		r := t.Runner.(globalRunner)
		if err := r.Run(t, opts); err != nil {
			return WrapWithTarget(err, t)
		}
		return nil

	case ChkKindEachComponent:
		c, ok := tgt.(*Component)
		if !ok {
			return WrappedErr{
				Target:       tgt,
				ActionTarget: t,
				Err:          fmt.Errorf("cannot check direct target %T with %v", tgt, t.Kind),
			}
		}
		r := t.Runner.(eachComponentRunner)

		for _, pop := range r.PopulatorsNeeded() {
			if c.RuntimeInfo().HasRun(pop) {
				continue
			}
			if err := pop.Run(c, opts, c.RuntimeInfo()); err != nil {
				err = WrapWithTarget(err, tgt)
				return err
			}
		}
		if err := r.Run(c, t, opts); err != nil {
			return WrapWithTarget(err, tgt)
		}
		return nil
	}

	return WrappedErr{
		Target:       tgt,
		ActionTarget: t,
		Err:          fmt.Errorf("checking direct target with %v not supported", t.Kind),
	}
}
