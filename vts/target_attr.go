package vts

import (
	"crypto/sha256"
	"errors"
	"fmt"

	"go.starlark.net/starlark"
)

// Attr is a target representing an attribute.
type Attr struct {
	Path   string
	Name   string
	Parent TargetRef
	Pos    *DefPosition

	Val starlark.Value
}

func (t *Attr) DefinedAt() *DefPosition {
	return t.Pos
}

func (t *Attr) IsClassTarget() bool {
	return false
}

func (t *Attr) TargetType() TargetType {
	return TargetAttr
}

func (t *Attr) Class() TargetRef {
	return t.Parent
}

func (t *Attr) GlobalPath() string {
	return t.Path
}

func (t *Attr) TargetName() string {
	return t.Name
}

func (t *Attr) Validate() error {
	if t.Parent.Target != nil {
		if _, ok := t.Parent.Target.(*AttrClass); !ok {
			return fmt.Errorf("parent is type %T, but must be attr_class", t.Parent.Target)
		}
	} else if t.Parent.Path == "" {
		return errors.New("no parent attr_class specified")
	}

	if cv, isComputedValue := t.Val.(*ComputedValue); isComputedValue {
		if err := cv.Validate(); err != nil {
			return fmt.Errorf("invalid computed value: %v", err)
		}
	}

	return nil
}

func (t *Attr) String() string {
	return fmt.Sprintf("attr<%s>", t.Name)
}

func (t *Attr) Freeze() {

}

func (t *Attr) Truth() starlark.Bool {
	return true
}

func (t *Attr) Type() string {
	return "attr"
}

func (t *Attr) Hash() (uint32, error) {
	h := sha256.Sum256([]byte(fmt.Sprintf("%p", t)))
	return uint32(uint32(h[0]) + uint32(h[1])<<8 + uint32(h[2])<<16 + uint32(h[3])<<24), nil
}

type computeEval func(attr *Attr, target Target, runInfo *ComputedValue, env *RunnerEnv) (starlark.Value, error)

// Value returns the value of the attribute, invoking any computation
// if necessary.
func (t *Attr) Value(parent Target, env *RunnerEnv, eval computeEval) (starlark.Value, error) {
	if cv, isComputed := t.Val.(*ComputedValue); isComputed {
		v, err := eval(t, parent, cv, env)
		if err != nil {
			return v, WrapWithComputedValue(err, cv)
		}
		return v, nil
	}
	return t.Val, nil
}
