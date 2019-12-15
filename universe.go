package ccr

import (
	"errors"
	"fmt"

	"github.com/twitchylinux/ccr/vts"
)

type ErrNotExists string

func (e ErrNotExists) Error() string {
	return fmt.Sprintf("target %q does not exist", string(e))
}

// Universe stores a tree representation of targets.
type Universe struct {
	// fqTargets enumerates all targets by their full path.
	fqTargets map[string]vts.GlobalTarget
	// allTargets contains a list of all known targets in enumeration order.
	allTargets []vts.GlobalTarget
}

func (u *Universe) makeTargetRef(from vts.TargetRef) (vts.TargetRef, error) {
	if from.Target != nil {
		return from, nil
	}
	if from.Path == "" {
		return vts.TargetRef{}, errors.New("cannot reference target with empty path")
	}
	if t, ok := u.fqTargets[from.Path]; ok {
		return vts.TargetRef{Target: t}, nil
	}
	return vts.TargetRef{}, ErrNotExists(from.Path)
}

// linkTarget updates references within the target to point to the target
// rather than the correct path.
func (u *Universe) linkTarget(t vts.Target) error {
	var err error
	switch n := t.(type) {
	case *vts.Component:
		for i := range n.Deps {
			if n.Deps[i], err = u.makeTargetRef(n.Deps[i]); err != nil {
				return err
			}
		}
		for i := range n.Details {
			if n.Details[i], err = u.makeTargetRef(n.Details[i]); err != nil {
				return err
			}
		}
		return nil

	case *vts.ResourceClass:
		for i := range n.Deps {
			if n.Deps[i], err = u.makeTargetRef(n.Deps[i]); err != nil {
				return err
			}
		}
		for i := range n.Checks {
			if n.Checks[i], err = u.makeTargetRef(n.Checks[i]); err != nil {
				return err
			}
		}
		return nil

	case *vts.Resource:
		if n.Parent, err = u.makeTargetRef(n.Parent); err != nil {
			return err
		}
		for i := range n.Deps {
			if n.Deps[i], err = u.makeTargetRef(n.Deps[i]); err != nil {
				return err
			}
		}
		for i := range n.Details {
			if n.Details[i], err = u.makeTargetRef(n.Details[i]); err != nil {
				return err
			}
		}
		return nil

	case *vts.Attr:
		if n.Parent, err = u.makeTargetRef(n.Parent); err != nil {
			return err
		}
		return nil

	case *vts.AttrClass:
		for i := range n.Checks {
			if n.Checks[i], err = u.makeTargetRef(n.Checks[i]); err != nil {
				return err
			}
		}
		return nil
	}

	return fmt.Errorf("linking failed: cannot handle target of type %T", t)
}

func (u *Universe) insertResolvedTarget(t vts.GlobalTarget) error {
	p := t.GlobalPath()
	if _, exists := u.fqTargets[p]; exists {
		return nil
	}
	u.fqTargets[p] = t
	u.allTargets = append(u.allTargets, t)
	return nil
}

func (u *Universe) resolveTarget(findOpts *FindOptions, t vts.Target) error {
	gt, ok := t.(vts.GlobalTarget)
	if ok {
		if err := u.insertResolvedTarget(gt); err != nil {
			return err
		}
	}

	if class, hasClass := gt.(vts.ClassedTarget); hasClass {
		if err := u.resolveRef(findOpts, class.Class()); err != nil {
			return err
		}
	}
	if deps, hasDeps := gt.(vts.DepTarget); hasDeps {
		for _, dep := range deps.Dependencies() {
			if err := u.resolveRef(findOpts, dep); err != nil {
				return err
			}
		}
	}
	if chks, hasCheckers := gt.(vts.CheckedTarget); hasCheckers {
		for _, c := range chks.Checkers() {
			if err := u.resolveRef(findOpts, c); err != nil {
				return err
			}
		}
	}
	if deets, hasDetails := gt.(vts.DetailedTarget); hasDetails {
		for _, attr := range deets.Attributes() {
			if err := u.resolveRef(findOpts, attr); err != nil {
				return err
			}
		}
	}
	if err := u.linkTarget(t); err != nil {
		return err
	}
	if err := t.Validate(); err != nil {
		return err // TODO: Plumb file/line numbers here somehow
	}
	return nil
}

func (u *Universe) resolveRef(findOpts *FindOptions, t vts.TargetRef) error {
	if t.Target != nil {
		return u.resolveTarget(findOpts, t.Target)
	}

	if _, ok := u.fqTargets[t.Path]; ok {
		return nil // Already known.
	}
	target, err := findOpts.Find(t.Path)
	if err != nil {
		return err
	}
	return u.resolveTarget(findOpts, target)
}

// Build constructs a fully-resolved tree of targets from those given.
func (u *Universe) Build(targets []vts.TargetRef, findOpts *FindOptions) error {
	for _, t := range targets {
		if err := u.resolveRef(findOpts, t); err != nil {
			return err
		}
	}
	return nil
}

// EnumeratedTargets returns an ordered list of all targets,
// in the order they were enumerated.
func (u *Universe) EnumeratedTargets() []vts.GlobalTarget {
	return u.allTargets
}

func NewUniverse() *Universe {
	return &Universe{
		allTargets: make([]vts.GlobalTarget, 0, 4096),
		fqTargets:  make(map[string]vts.GlobalTarget, 4096),
	}
}
