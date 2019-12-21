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
	return ac.Run(attr, opts)
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
	return ac.Run(r, opts)
}

// RunCheckedTarget is invoked when running checks directly on the target
// on which they are defined.
func (t *Checker) RunCheckedTarget(tgt CheckedTarget, opts *RunnerEnv) error {
	if t.Kind != ChkKindEachComponent {
		return fmt.Errorf("RunCheckedTarget() called on non-component checker %v", t.Kind)
	}
	return nil
}
