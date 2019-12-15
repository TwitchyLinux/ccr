package vts

import "fmt"

// Component is a target representing a component.
type Component struct {
	Path string
	Name string

	Details []TargetRef
	Deps    []TargetRef
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

func (t *Component) Attributes() []TargetRef {
	return t.Details
}

func (t *Component) Validate() error {
	for i, deet := range t.Details {
		if _, ok := deet.Target.(*Attr); !ok {
			return fmt.Errorf("details[%d] is type %T, but must be attr", i, deet.Target)
		}
	}
	for i, dep := range t.Deps {
		_, component := dep.Target.(*Component)
		_, resource := dep.Target.(*Resource)
		if !component && !resource {
			return fmt.Errorf("deps[%d] is type %T, but must be resource or component", i, dep.Target)
		}
	}
	return nil
}
