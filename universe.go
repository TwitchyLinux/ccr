// Package ccr works with component contracts.
package ccr

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/twitchylinux/ccr/vts"
	"github.com/twitchylinux/ccr/vts/common"
	"go.starlark.net/starlark"
)

var ErrNotBuilt = errors.New("universe must be built first")

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
	// classedTargets enumerates the targets which chain to a specific parent.
	classedTargets map[vts.Target][]vts.GlobalTarget

	resolved bool
	logger   opTrack
	cache    *Cache
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
				return u.logger.Error(MsgBadRef, vts.WrapWithTarget(err, t))
			}
		}
		for i := range n.Details {
			if n.Details[i], err = u.makeTargetRef(n.Details[i]); err != nil {
				return u.logger.Error(MsgBadRef, vts.WrapWithTarget(err, t))
			}
		}
		for i := range n.Checks {
			if n.Checks[i], err = u.makeTargetRef(n.Checks[i]); err != nil {
				return u.logger.Error(MsgBadRef, vts.WrapWithTarget(err, t))
			}
		}
		return nil

	case *vts.ResourceClass:
		for i := range n.Deps {
			if n.Deps[i], err = u.makeTargetRef(n.Deps[i]); err != nil {
				return u.logger.Error(MsgBadRef, vts.WrapWithTarget(err, t))
			}
		}
		for i := range n.Checks {
			if n.Checks[i], err = u.makeTargetRef(n.Checks[i]); err != nil {
				return u.logger.Error(MsgBadRef, vts.WrapWithTarget(err, t))
			}
		}
		return nil

	case *vts.Resource:
		if n.Parent, err = u.makeTargetRef(n.Parent); err != nil {
			return u.logger.Error(MsgBadRef, vts.WrapWithTarget(err, t))
		}
		if n.Source != nil {
			tmp, err := u.makeTargetRef(*n.Source)
			if err != nil {
				return u.logger.Error(MsgBadRef, vts.WrapWithTarget(err, t))
			}
			n.Source = &tmp
		}
		for i := range n.Deps {
			if n.Deps[i], err = u.makeTargetRef(n.Deps[i]); err != nil {
				return u.logger.Error(MsgBadRef, vts.WrapWithTarget(err, t))
			}
		}
		for i := range n.Details {
			if n.Details[i], err = u.makeTargetRef(n.Details[i]); err != nil {
				return u.logger.Error(MsgBadRef, vts.WrapWithTarget(err, t))
			}
		}
		return nil

	case *vts.Attr:
		if n.Parent, err = u.makeTargetRef(n.Parent); err != nil {
			return u.logger.Error(MsgBadRef, vts.WrapWithTarget(err, t))
		}
		return nil

	case *vts.AttrClass:
		for i := range n.Checks {
			if n.Checks[i], err = u.makeTargetRef(n.Checks[i]); err != nil {
				return u.logger.Error(MsgBadRef, vts.WrapWithTarget(err, t))
			}
		}
		return nil

	case *vts.Generator:
		for i := range n.Inputs {
			if n.Inputs[i], err = u.makeTargetRef(n.Inputs[i]); err != nil {
				return u.logger.Error(MsgBadRef, vts.WrapWithTarget(err, t))
			}
		}
		return nil

	case *vts.Checker, *vts.Puesdo:
		return nil

	case *vts.Toolchain:
		for i := range n.Deps {
			if n.Deps[i], err = u.makeTargetRef(n.Deps[i]); err != nil {
				return u.logger.Error(MsgBadRef, vts.WrapWithTarget(err, t))
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

	class, hasClass := gt.(vts.ClassedTarget)
	if hasClass {
		if err := u.resolveRef(findOpts, class.Class()); err != nil {
			return err
		}
	}
	if inputs, hasInputs := gt.(vts.InputTarget); hasInputs {
		for _, inp := range inputs.NeedInputs() {
			if err := u.resolveRef(findOpts, inp); err != nil {
				return err
			}
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
	if st, isSrcdTarget := gt.(vts.SourcedTarget); isSrcdTarget && st.Src() != nil {
		if err := u.resolveRef(findOpts, *st.Src()); err != nil {
			return err
		}
	}
	if err := u.linkTarget(t); err != nil {
		return err
	}

	if err := t.Validate(); err != nil {
		return u.logger.Error(MsgBadDef, vts.WrapWithTarget(err, t))
	}
	// After linking, a target which has a parent will reference the parent. We
	// track all instances of a class to simplify resolving inputs of a class.
	if hasClass {
		classTarget := class.Class().Target.(vts.Target)
		u.classedTargets[classTarget] = append(u.classedTargets[classTarget], gt)
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
		return u.logger.Error(MsgBadFind, vts.WrapWithTarget(err, target))
	}
	return u.resolveTarget(findOpts, target)
}

// Build constructs a fully-resolved tree of targets from those given, and
// applies VTS-level validation against them.
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

var errNoAttr = errors.New("attr not specified")

func determineAttrValue(t vts.Target, cls *vts.AttrClass) (starlark.Value, error) {
	dt, ok := t.(vts.DetailedTarget)
	if !ok {
		return nil, vts.WrapWithTarget(fmt.Errorf("no details available on target %T", t), t)
	}
	for _, attr := range dt.Attributes() {
		if attr.Target == nil {
			return nil, vts.WrapWithTarget(fmt.Errorf("unresolved target reference: %q", attr.Path), t)
		}
		a := attr.Target.(*vts.Attr)
		if a.Parent.Target == nil {
			return nil, vts.WrapWithTarget(fmt.Errorf("unresolved target reference: %q", a.Parent.Path), t)
		}
		if class := a.Parent.Target.(*vts.AttrClass); class.GlobalPath() == cls.Path {
			return a.Value, nil
		}
	}

	return nil, errNoAttr
}

func determinePath(t vts.Target) (string, error) {
	v, err := determineAttrValue(t, common.PathClass)
	if err != nil {
		return "", err
	}
	if s, ok := v.(starlark.String); ok {
		return filepath.Clean(string(s)), nil
	}
	return "", vts.WrapWithTarget(fmt.Errorf("bad type for path: want string, got %T", v), t)
}

func determineMode(t vts.Target) (os.FileMode, error) {
	v, err := determineAttrValue(t, common.ModeClass)
	if err != nil {
		return 0, err
	}
	if s, ok := v.(starlark.String); ok {
		mode, err := strconv.ParseInt(string(s), 8, 64)
		return os.FileMode(mode), err
	}
	return 0, vts.WrapWithTarget(fmt.Errorf("bad type for mode: want string, got %T", v), t)
}

// NewUniverse constructs an empty universe.
func NewUniverse(logger opTrack, cache *Cache) *Universe {
	if logger == nil {
		logger = &consoleOpTrack{}
	}
	return &Universe{
		cache:          cache,
		allTargets:     make([]vts.GlobalTarget, 0, 4096),
		fqTargets:      make(map[string]vts.GlobalTarget, 4096),
		classedTargets: make(map[vts.Target][]vts.GlobalTarget, 2048),
		logger:         logger,
	}
}
