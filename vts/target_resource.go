package vts

import (
	"errors"
	"fmt"
)

// Resource is a target representing a resource.
type Resource struct {
	Path    string
	Name    string
	Parent  TargetRef
	Details []TargetRef
	Deps    []TargetRef
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

func (t *Resource) Attributes() []TargetRef {
	return t.Details
}

func (t *Resource) Validate() error {
	if t.Parent.Target != nil {
		if _, ok := t.Parent.Target.(*ResourceClass); !ok {
			return fmt.Errorf("parent is type %T, but must be resource_class", t.Parent.Target)
		}
	} else if t.Parent.Path == "" {
		return errors.New("no parent attr_class specified")
	}
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
