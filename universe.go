// Package ccr works with component contracts.
package ccr

import (
	"errors"
	"fmt"

	"github.com/twitchylinux/ccr/vts"
)

// ErrNotExists is returned if a referenced target could not be found.
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

	resolved bool
	logger   opTrack
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
// object rather than reference their path.
func (u *Universe) linkTarget(t vts.Target) error {
	var err error
	switch n := t.(type) {
	case *vts.Component:
		for i := range n.Deps {
			if n.Deps[i], err = u.makeTargetRef(n.Deps[i]); err != nil {
				return u.logger.Error(t, MsgBadRef, err)
			}
		}
		for i := range n.Details {
			if n.Details[i], err = u.makeTargetRef(n.Details[i]); err != nil {
				return u.logger.Error(t, MsgBadRef, err)
			}
		}
		for i := range n.Checks {
			if n.Checks[i], err = u.makeTargetRef(n.Checks[i]); err != nil {
				return u.logger.Error(t, MsgBadRef, err)
			}
		}
		return nil

	case *vts.ResourceClass:
		for i := range n.Deps {
			if n.Deps[i], err = u.makeTargetRef(n.Deps[i]); err != nil {
				return u.logger.Error(t, MsgBadRef, err)
			}
		}
		for i := range n.Checks {
			if n.Checks[i], err = u.makeTargetRef(n.Checks[i]); err != nil {
				return u.logger.Error(t, MsgBadRef, err)
			}
		}
		return nil

	case *vts.Resource:
		if n.Parent, err = u.makeTargetRef(n.Parent); err != nil {
			return u.logger.Error(t, MsgBadRef, err)
		}
		for i := range n.Deps {
			if n.Deps[i], err = u.makeTargetRef(n.Deps[i]); err != nil {
				return u.logger.Error(t, MsgBadRef, err)
			}
		}
		for i := range n.Details {
			if n.Details[i], err = u.makeTargetRef(n.Details[i]); err != nil {
				return u.logger.Error(t, MsgBadRef, err)
			}
		}
		return nil

	case *vts.Attr:
		if n.Parent, err = u.makeTargetRef(n.Parent); err != nil {
			return u.logger.Error(t, MsgBadRef, err)
		}
		return nil

	case *vts.AttrClass:
		for i := range n.Checks {
			if n.Checks[i], err = u.makeTargetRef(n.Checks[i]); err != nil {
				return u.logger.Error(t, MsgBadRef, err)
			}
		}
		return nil

	case *vts.Checker:
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
		return u.logger.Error(target, MsgBadFind, err)
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
	u.resolved = true
	return nil
}

// EnumeratedTargets returns an ordered list of all targets,
// in the order they were enumerated.
func (u *Universe) EnumeratedTargets() []vts.GlobalTarget {
	return u.allTargets
}

type targetSet map[vts.Target]struct{}

// Check runs the checkers for all reachable targets against the system
// in basePath.
func (u *Universe) Check(targets []vts.TargetRef, basePath string) error {
	if !u.resolved {
		return errors.New("universe must be resolved first")
	}

	var (
		evaluatedTargets = make(targetSet, 4096)
		opts             = vts.CheckerOpts{
			Dir: basePath,
		}
	)
	for _, t := range targets {
		target := t.Target
		if target == nil {
			var ok bool
			target, ok = u.fqTargets[t.Path]
			if !ok {
				return ErrNotExists(t.Path)
			}
		}
		if err := u.checkTarget(target, &opts, evaluatedTargets); err != nil {
			return err
		}
	}
	return nil
}

func (u *Universe) checkTarget(t vts.Target, opts *vts.CheckerOpts, checked targetSet) error {
	if _, checked := checked[t]; checked {
		return nil
	}
	checked[t] = struct{}{}

	// Check dependencies first.
	if deps, hasDeps := t.(vts.DepTarget); hasDeps {
		for _, dep := range deps.Dependencies() {
			if err := u.checkTarget(dep.Target, opts, checked); err != nil {
				return err
			}
		}
	}
	// Validate attributes by recursing.
	if deets, hasDetails := t.(vts.DetailedTarget); hasDetails {
		for _, attr := range deets.Attributes() {
			if err := u.checkTarget(attr.Target, opts, checked); err != nil {
				return err
			}
		}
	}

	// Validate checkers defined on a class target if applicable.
	if class, hasClass := t.(vts.ClassedTarget); hasClass {
		switch n := class.Class().Target.(type) {
		case *vts.ResourceClass:
			if err := n.RunCheckers(t.(*vts.Resource), opts); err != nil {
				return u.logger.Error(t, MsgFailedCheck, err)
			}
		case *vts.AttrClass:
			if err := n.RunCheckers(t.(*vts.Attr), opts); err != nil {
				return u.logger.Error(t, MsgFailedCheck, err)
			}
		}
	}

	// Finally, validate any checkers on the target itself.
	if !t.IsClassTarget() {
		if n, hasChecks := t.(vts.CheckedTarget); hasChecks {
			for _, c := range n.Checkers() {
				if err := c.Target.(*vts.Checker).RunCheckedTarget(n, opts); err != nil {
					return u.logger.Error(t, MsgFailedCheck, err)
				}
			}
		}
	}
	return nil
}

// NewUniverse constructs an empty universe.
func NewUniverse(logger opTrack) *Universe {
	if logger == nil {
		logger = &consoleOpTrack{}
	}
	return &Universe{
		allTargets: make([]vts.GlobalTarget, 0, 4096),
		fqTargets:  make(map[string]vts.GlobalTarget, 4096),
		logger:     logger,
	}
}
