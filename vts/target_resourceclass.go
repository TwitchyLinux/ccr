package vts

import "fmt"

// ResourceClass is a target representing a resource class.
type ResourceClass struct {
	Path string
	Name string
	Pos  *DefPosition

	Deps   []TargetRef
	Checks []TargetRef
}

func (t *ResourceClass) DefinedAt() *DefPosition {
	return t.Pos
}

func (t *ResourceClass) IsClassTarget() bool {
	return true
}

func (t *ResourceClass) TargetType() TargetType {
	return TargetResourceClass
}

func (t *ResourceClass) GlobalPath() string {
	return t.Path
}

func (t *ResourceClass) TargetName() string {
	return t.Name
}

func (t *ResourceClass) Dependencies() []TargetRef {
	return t.Deps
}

func (t *ResourceClass) Checkers() []TargetRef {
	return t.Checks
}

func (t *ResourceClass) Validate() error {
	if err := validateDeps(t.Deps, false); err != nil {
		return err
	}
	return nil
}

// RunCheckers runs checkers on a resource, in the context of this
// resource class.
func (t *ResourceClass) RunCheckers(r *Resource, opts *RunnerEnv) error {
	for i, c := range t.Checks {
		if c.Target == nil {
			return fmt.Errorf("check[%d] is not resolved: %q", i, c.Path)
		}
		if err := c.Target.(*Checker).RunResource(r, opts); err != nil {
			return err
		}
	}
	return nil
}
