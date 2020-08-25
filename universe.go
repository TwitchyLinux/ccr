// Package ccr works with component contracts.
package ccr

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/twitchylinux/ccr/cache"
	"github.com/twitchylinux/ccr/log"
	"github.com/twitchylinux/ccr/proc"
	"github.com/twitchylinux/ccr/vts"
	"github.com/twitchylinux/ccr/vts/common"
	"go.starlark.net/starlark"
	"go.starlark.net/syntax"
	"gopkg.in/src-d/go-billy.v4/osfs"
)

var ErrNotBuilt = errors.New("universe must be built first")

// ErrNotExists is returned if a referenced target could not be found.
type ErrNotExists string

func (e ErrNotExists) Error() string {
	return fmt.Sprintf("target %q does not exist", string(e))
}

type opTrack interface {
	Error(category log.MsgCategory, err error) error
	Warning(category log.MsgCategory, message string)
	Info(category log.MsgCategory, message string)
	IsInteractive() bool
	Stdout() io.Writer
	Stderr() io.Writer
}

// Universe stores a tree representation of targets.
type Universe struct {
	// fqTargets enumerates all targets by their full path.
	fqTargets map[string]vts.GlobalTarget
	// allTargets contains a list of all known targets in enumeration order.
	allTargets []vts.GlobalTarget
	// classedTargets enumerates the targets which chain to a specific parent.
	classedTargets map[vts.Target][]vts.GlobalTarget
	// pathTargets enumerates the targets which had a specific path attribute.
	pathTargets map[string]vts.Target
	// globalCheckers enumerates all global checkers.
	globalCheckers []*vts.Checker

	resolved bool
	logger   opTrack
	cache    *cache.Cache
}

func (u *Universe) makeTargetRef(from vts.TargetRef) (vts.TargetRef, error) {
	for i, _ := range from.Constraints {
		var err error
		if from.Constraints[i].Meta, err = u.makeTargetRef(from.Constraints[i].Meta); err != nil {
			return vts.TargetRef{}, err
		}
	}
	if from.Target != nil {
		return from, nil
	}
	if from.Path == "" {
		return vts.TargetRef{}, errors.New("cannot reference target with empty path")
	}
	if t, ok := u.fqTargets[from.Path]; ok {
		return vts.TargetRef{Target: t, Constraints: from.Constraints}, nil
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
				return u.logger.Error(log.MsgBadRef, vts.WrapWithTarget(err, t))
			}
		}
		for i := range n.Details {
			if n.Details[i], err = u.makeTargetRef(n.Details[i]); err != nil {
				return u.logger.Error(log.MsgBadRef, vts.WrapWithTarget(err, t))
			}
		}
		for i := range n.Checks {
			if n.Checks[i], err = u.makeTargetRef(n.Checks[i]); err != nil {
				return u.logger.Error(log.MsgBadRef, vts.WrapWithTarget(err, t))
			}
		}
		return nil

	case *vts.ResourceClass:
		for i := range n.Deps {
			if n.Deps[i], err = u.makeTargetRef(n.Deps[i]); err != nil {
				return u.logger.Error(log.MsgBadRef, vts.WrapWithTarget(err, t))
			}
		}
		for i := range n.Checks {
			if n.Checks[i], err = u.makeTargetRef(n.Checks[i]); err != nil {
				return u.logger.Error(log.MsgBadRef, vts.WrapWithTarget(err, t))
			}
		}
		return nil

	case *vts.Resource:
		if n.Parent, err = u.makeTargetRef(n.Parent); err != nil {
			return u.logger.Error(log.MsgBadRef, vts.WrapWithTarget(err, t))
		}
		if n.Source != nil {
			tmp, err := u.makeTargetRef(*n.Source)
			if err != nil {
				return u.logger.Error(log.MsgBadRef, vts.WrapWithTarget(err, t))
			}
			n.Source = &tmp
		}
		for i := range n.Deps {
			if n.Deps[i], err = u.makeTargetRef(n.Deps[i]); err != nil {
				return u.logger.Error(log.MsgBadRef, vts.WrapWithTarget(err, t))
			}
		}
		for i := range n.Details {
			if n.Details[i], err = u.makeTargetRef(n.Details[i]); err != nil {
				return u.logger.Error(log.MsgBadRef, vts.WrapWithTarget(err, t))
			}
		}
		return nil

	case *vts.Attr:
		if n.Parent, err = u.makeTargetRef(n.Parent); err != nil {
			return u.logger.Error(log.MsgBadRef, vts.WrapWithTarget(err, t))
		}
		return nil

	case *vts.AttrClass:
		for i := range n.Checks {
			if n.Checks[i], err = u.makeTargetRef(n.Checks[i]); err != nil {
				return u.logger.Error(log.MsgBadRef, vts.WrapWithTarget(err, t))
			}
		}
		return nil

	case *vts.Generator:
		for i := range n.Inputs {
			if n.Inputs[i], err = u.makeTargetRef(n.Inputs[i]); err != nil {
				return u.logger.Error(log.MsgBadRef, vts.WrapWithTarget(err, t))
			}
		}
		return nil

	case *vts.Checker, *vts.Puesdo:
		return nil

	case *vts.Toolchain:
		for i := range n.Deps {
			if n.Deps[i], err = u.makeTargetRef(n.Deps[i]); err != nil {
				return u.logger.Error(log.MsgBadRef, vts.WrapWithTarget(err, t))
			}
		}
		for i := range n.Details {
			if n.Details[i], err = u.makeTargetRef(n.Details[i]); err != nil {
				return u.logger.Error(log.MsgBadRef, vts.WrapWithTarget(err, t))
			}
		}
		return nil

	case *vts.Build:
		for i := range n.HostDeps {
			if n.HostDeps[i], err = u.makeTargetRef(n.HostDeps[i]); err != nil {
				return u.logger.Error(log.MsgBadRef, vts.WrapWithTarget(err, t))
			}
		}
		for i := range n.Injections {
			if n.Injections[i], err = u.makeTargetRef(n.Injections[i]); err != nil {
				return u.logger.Error(log.MsgBadRef, vts.WrapWithTarget(err, t))
			}
		}
		out := make(map[string]vts.TargetRef, len(n.PatchIns))
		for k, v := range n.PatchIns {
			if out[k], err = u.makeTargetRef(v); err != nil {
				return u.logger.Error(log.MsgBadRef, vts.WrapWithTarget(err, t))
			}
		}
		n.PatchIns = out
		return nil

	case *vts.Sieve:
		for i := range n.Inputs {
			if n.Inputs[i], err = u.makeTargetRef(n.Inputs[i]); err != nil {
				return u.logger.Error(log.MsgBadRef, vts.WrapWithTarget(err, t))
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
	if deps, hasHostDeps := gt.(vts.HostDepTarget); hasHostDeps {
		for _, dep := range deps.HostDependencies() {
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
		return u.logger.Error(log.MsgBadDef, vts.WrapWithTarget(err, t))
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
		if be, isBuildErr := err.(buildErr); isBuildErr {
			if synErr, isSynErr := be.err.(syntax.Error); isSynErr {
				err = vts.WrapWithPosition(errors.New(synErr.Msg), &vts.DefPosition{
					Path:  be.path,
					Frame: starlark.CallFrame{Pos: synErr.Pos},
				})
				return u.logger.Error(log.MsgBadDef, vts.WrapWithTarget(err, target))
			}
		}
		return u.logger.Error(log.MsgBadFind, vts.WrapWithTarget(err, target))
	}
	return u.resolveTarget(findOpts, target)
}

func (u *Universe) makeEnv(basePath string) *vts.RunnerEnv {
	return &vts.RunnerEnv{
		Dir:      basePath,
		FS:       osfs.New(basePath),
		Universe: &runtimeResolver{u, map[string]interface{}{}},
	}
}

// Build constructs a fully-resolved tree of targets from those given, and
// applies VTS-level validation against them.
func (u *Universe) Build(targets []vts.TargetRef, findOpts *FindOptions, basePath string) error {
	for _, t := range targets {
		if err := u.resolveRef(findOpts, t); err != nil {
			return err
		}
	}

	// Track special targets separately.
	for _, t := range u.allTargets {
		// Track all global checkers.
		if chkr, isChecker := t.(*vts.Checker); isChecker && chkr.Kind == vts.ChkKindGlobal {
			u.globalCheckers = append(u.globalCheckers, chkr)
		}

		if _, isDetailed := t.(vts.DetailedTarget); !isDetailed {
			continue
		}
		// Track targets that declare a path.
		path, err := determinePath(t, u.makeEnv(basePath))
		if err == errNoAttr {
			continue
		}
		if err != nil {
			return u.logger.Error(log.MsgBadDef, vts.WrapWithTarget(err, t))
		}

		if e, exists := u.pathTargets[path]; exists {
			return u.logger.Error(log.MsgBadDef, vts.WrapWithPath(
				vts.WrapWithActionTarget(vts.WrapWithTarget(errors.New("multiple targets declared the same path"), e), t), path))
		}
		u.pathTargets[path] = t
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

func determineAttrValue(t vts.Target, cls *vts.AttrClass, env *vts.RunnerEnv) (starlark.Value, error) {
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
			return a.Value(t, env, proc.EvalComputedAttribute)
		}
	}

	return nil, errNoAttr
}

func determinePath(t vts.Target, env *vts.RunnerEnv) (string, error) {
	v, err := determineAttrValue(t, common.PathClass, env)
	if err != nil {
		return "", err
	}
	if s, ok := v.(starlark.String); ok {
		return filepath.Clean(string(s)), nil
	}
	return "", vts.WrapWithTarget(fmt.Errorf("bad type for path: want string, got %T", v), t)
}

func (u *Universe) query(basePath, target, attr string, byParent bool) (starlark.Value, error) {
	if !u.resolved {
		return starlark.None, ErrNotBuilt
	}
	t, ok := u.fqTargets[target]
	if !ok {
		return starlark.None, ErrNotExists(target)
	}
	detailedTarget, ok := t.(vts.DetailedTarget)
	if !ok {
		return starlark.None, fmt.Errorf("no details on target %q", target)
	}
	for _, at := range detailedTarget.Attributes() {
		if at.Target == nil {
			return starlark.None, fmt.Errorf("nil attribute target on %v", at)
		}

		attrib := at.Target.(*vts.Attr)
		if (!byParent && attrib.Name == attr) || (byParent && attrib.Parent.Target.(*vts.AttrClass).Path == attr) {
			return attrib.Value(detailedTarget, u.makeEnv(basePath), proc.EvalComputedAttribute)
		}
	}

	return starlark.None, nil
}

func (u *Universe) QueryByName(basePath, target, attr string) (starlark.Value, error) {
	return u.query(basePath, target, attr, false)
}

func (u *Universe) QueryByClass(basePath, target, parent string) (starlark.Value, error) {
	return u.query(basePath, target, parent, true)
}

// GetTarget returns the target with the specified name.
func (u *Universe) GetTarget(name string) vts.GlobalTarget {
	return u.fqTargets[name]
}

func (u *Universe) TargetRollupHash(name string) ([]byte, error) {
	t, ok := u.fqTargets[name]
	if !ok {
		return nil, os.ErrNotExist
	}
	rt, ok := t.(vts.ReproducibleTarget)
	if !ok {
		return nil, fmt.Errorf("target %T cannot be hashed", t)
	}
	return rt.RollupHash(u.makeEnv("/"), proc.EvalComputedAttribute)
}

// NewUniverse constructs an empty universe.
func NewUniverse(logger opTrack, cache *cache.Cache) *Universe {
	if logger == nil {
		logger = &log.Console{}
	}
	return &Universe{
		cache:          cache,
		allTargets:     make([]vts.GlobalTarget, 0, 4096),
		fqTargets:      make(map[string]vts.GlobalTarget, 4096),
		classedTargets: make(map[vts.Target][]vts.GlobalTarget, 2048),
		pathTargets:    make(map[string]vts.Target, 1024),
		logger:         logger,
	}
}
