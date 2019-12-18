package vts

import (
	"crypto/sha256"
	"fmt"

	"go.starlark.net/starlark"
)

type CheckerKind string

// Valid checker kinds.
const (
	ChkKindEachResource CheckerKind = "each_resource"
	ChkKindEachAttr     CheckerKind = "each_attr"
)

// Checker is a target describing a check on a target or the system.
type Checker struct {
	Path string
	Name string
	Kind CheckerKind

	Runner starlark.Value
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

func (t *Checker) RunAttr(attr *Attr, opts *CheckerOpts) error {
	if t.Kind != ChkKindEachAttr {
		return fmt.Errorf("checker %v cannot be used to check attr", t.Kind)
	}
	ac, ok := t.Runner.(eachAttrRunner)
	if !ok {
		return fmt.Errorf("runner %v cannot be used to check attr", t.Runner)
	}
	return ac.Run(attr, opts)
}

func (t *Checker) RunResource(r *Resource, opts *CheckerOpts) error {
	if t.Kind != ChkKindEachResource {
		return fmt.Errorf("checker %v cannot be used to check resource", t.Kind)
	}
	ac, ok := t.Runner.(eachResourceRunner)
	if !ok {
		return fmt.Errorf("runner %v cannot be used to check resource", t.Runner)
	}
	return ac.Run(r, opts)
}
