package vts

import (
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

func (t *Resource) Validate() error {
	if t.Parent.Target != nil {
		if _, ok := t.Parent.Target.(*ResourceClass); !ok {
			return fmt.Errorf("parent is type %T, but must be resource_class", t.Parent.Target)
		}
	} else if t.Parent.Path == "" {
		return errors.New("no parent attr_class specified")
	}

	if err := validateDetails(t.Details); err != nil {
		return err
	}
	if err := validateDeps(t.Deps); err != nil {
		return err
	}
	if t.Source != nil {
		if err := validateSource(*t.Source, t); err != nil {
			return err
		}
	}
	return nil
}
