// Package vts contains CCR's virtual target system.
package vts

import "fmt"

type TargetType uint8

// Valid target types.
const (
	// TargetEmpty represents an invalid target.
	TargetEmpty TargetType = iota
	// TargetComponent represents a component target.
	TargetComponent
	// TargetResource represents a resource target.
	TargetResource
	// TargetResourceClass represents a resource class target.
	TargetResourceClass
	// TargetAttr represents an attribute target.
	TargetAttr
	// TargetAttrClass represents an attribute class target.
	TargetAttrClass
	// TargetChecker represents a checker target.
	TargetChecker
)

func (t TargetType) String() string {
	switch t {
	case TargetComponent:
		return "component"
	case TargetResource:
		return "resource"
	case TargetResourceClass:
		return "resource_class"
	case TargetAttr:
		return "attr"
	case TargetAttrClass:
		return "attr_class"
	case TargetChecker:
		return "checker"
	default:
		return fmt.Sprintf("TargetType<%d>", int(t))
	}
}

// Target describes a node, such as a resource or component, that
// participates in the the web of nodes declaring a system.
type Target interface {
	IsClassTarget() bool
	TargetType() TargetType
	Validate() error
}

// DepTarget describes a node which depends on other nodes.
type DepTarget interface {
	Target
	Dependencies() []TargetRef
}

// CheckedTarget describes a node which has checkers attached.
type CheckedTarget interface {
	Target
	Checkers() []TargetRef
}

// DetailedTarget describes a node which has details attached.
type DetailedTarget interface {
	Target
	Attributes() []TargetRef
}

// ClassedTarget describes a node which represents an instance of a class node.
type ClassedTarget interface {
	Target
	Class() TargetRef
}

// GlobalTarget describes a node which has an absolute path.
type GlobalTarget interface {
	Target
	GlobalPath() string
	TargetName() string
}

// TargetRef refers to another target either by path or by
// a pointer to its object.
type TargetRef struct {
	Path   string
	Target Target
}

func validateDeps(deps []TargetRef) error {
	for i, dep := range deps {
		_, component := dep.Target.(*Component)
		_, resource := dep.Target.(*Resource)
		if !component && !resource {
			return fmt.Errorf("deps[%d] is type %T, but must be resource or component", i, dep.Target)
		}
	}
	return nil
}

func validateDetails(details []TargetRef) error {
	for i, deet := range details {
		if _, ok := deet.Target.(*Attr); !ok {
			return fmt.Errorf("details[%d] is type %T, but must be attr", i, deet.Target)
		}
	}
	return nil
}
