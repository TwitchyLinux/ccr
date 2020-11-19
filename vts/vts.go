// Package vts contains CCR's virtual target system.
package vts

import (
	"errors"
	"fmt"

	"go.starlark.net/starlark"
)

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
	// TargetGenerator represents a generator target.
	TargetGenerator
	// TargetToolchain represents a description of a host toolchain.
	TargetToolchain
	// TargetPuesdo represents a special-case, generated target.
	TargetPuesdo
	// TargetBuild represents a build target.
	TargetBuild
	// TargetSieve represents a union/filter of other generator targets.
	TargetSieve
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
	case TargetGenerator:
		return "generator"
	case TargetToolchain:
		return "toolchain"
	case TargetPuesdo:
		return "puesdo"
	case TargetBuild:
		return "build"
	case TargetSieve:
		return "sieve"
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
	DefinedAt() *DefPosition
}

// DepTarget describes a node which depends on other nodes.
type DepTarget interface {
	Target
	Dependencies() []TargetRef
}

// HostDepTarget describes a node which depends on stuff on the host.
type HostDepTarget interface {
	Target
	HostDependencies() []TargetRef
}

// InputTarget describes a node which needs inputs from other nodes.
type InputTarget interface {
	Target
	NeedInputs() []TargetRef
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

// SourcedTarget describes a node whose implementation may be generated
// by a generator.
type SourcedTarget interface {
	Target
	Src() *TargetRef
}

// ReproducibleTarget describes a node where evaluations in the same
// environment and with the same hash will produce the same outputs.
type ReproducibleTarget interface {
	Target
	RollupHash(env *RunnerEnv, eval computeEval) ([]byte, error)
}

// UsingRootFSTarget describes a node whose build should use
// the referenced build artifacts as its chroot filesystem.
type UsingRootFSTarget interface {
	Target
	UsingRootFS() *TargetRef
}

// TargetRef refers to another target either by path or by
// a pointer to its object.
type TargetRef struct {
	Path        string
	Target      Target
	Constraints []RefConstraint
}

// RefConstraint describes a constraint on a referenced target.
// For instance, a constraint intended to check the semantic version
// of a target could have Meta == common.SemverClass, Params =
// ["1.2.3"], and Eval pointing to an evaluator which compares the two
// and returns an error or not.
type RefConstraint struct {
	Meta   TargetRef
	Params []starlark.Value
	Eval   ConstraintEvaluator
}

type ConstraintEvaluator interface {
	Check(env *RunnerEnv, val starlark.Value) error
}

func validateDeps(deps []TargetRef, allowConstraints bool) error {
	for i, dep := range deps {
		_, component := dep.Target.(*Component)
		_, resource := dep.Target.(*Resource)
		_, toolchain := dep.Target.(*Toolchain)
		if !component && !resource && !toolchain {
			return fmt.Errorf("deps[%d] is type %T, but must be resource or component", i, dep.Target)
		}

		if !allowConstraints && len(dep.Constraints) > 0 {
			return fmt.Errorf("deps[%d]: constraints are not valid here", i)
		}
	}
	return nil
}

func validateDetails(details []TargetRef) error {
	seenExclusiveClasses := map[*AttrClass]*Attr{}
	for i, deet := range details {
		a, ok := deet.Target.(*Attr)
		if !ok {
			return fmt.Errorf("details[%d] is type %T, but must be attr", i, deet.Target)
		}
		if parent := a.Parent.Target.(*AttrClass); !parent.Repeatable {
			if _, alreadySeen := seenExclusiveClasses[parent]; alreadySeen {
				return fmt.Errorf("duplicate attributes with non-repeatable class %q", parent.GlobalPath())
			}
			seenExclusiveClasses[parent] = a
		}
		if len(deet.Constraints) > 0 {
			return fmt.Errorf("details[%d]: constraints are not valid here", i)
		}
	}
	return nil
}

func validateSource(src TargetRef, parent Target) error {
	if src.Path == "" && src.Target == nil {
		return errors.New("source defined but no target or path present in reference")
	}
	if src.Target != nil {
		switch n := src.Target.(type) {
		case *Build:
		case *Generator:
		case *Sieve:
		case *Puesdo:
			switch n.Kind {
			case FileRef:
			case DebRef:
			default:
				return fmt.Errorf("puesdo target %v cannot be used as a source", n.Kind)
			}
		default:
			return fmt.Errorf("target of type %T cannot be used as a source", src.Target)
		}
	}
	if len(src.Constraints) > 0 {
		return errors.New("invalid source: constraints cannot be specified here")
	}
	return nil
}

// DefPosition describes where a target was defined.
type DefPosition struct {
	Path  string
	Frame starlark.CallFrame
}
