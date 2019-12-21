package vts

import "fmt"

// AttrClass is a target representing an attribute class.
type AttrClass struct {
	Path   string
	Name   string
	Pos    *DefPosition
	Checks []TargetRef
}

func (t *AttrClass) DefinedAt() *DefPosition {
	return t.Pos
}

func (t *AttrClass) IsClassTarget() bool {
	return true
}

func (t *AttrClass) TargetType() TargetType {
	return TargetAttrClass
}

func (t *AttrClass) GlobalPath() string {
	return t.Path
}

func (t *AttrClass) TargetName() string {
	return t.Name
}

func (t *AttrClass) Checkers() []TargetRef {
	return t.Checks
}

func (t *AttrClass) Validate() error {
	return nil
}

// RunCheckers runs checkers on an attribute, in the context of
// this attribute class.
func (t *AttrClass) RunCheckers(attr *Attr, opts *RunnerEnv) error {
	for i, c := range t.Checks {
		if c.Target == nil {
			return fmt.Errorf("check[%d] is not resolved: %q", i, c.Path)
		}
		if err := c.Target.(*Checker).RunAttr(attr, opts); err != nil {
			return err
		}
	}
	return nil
}
