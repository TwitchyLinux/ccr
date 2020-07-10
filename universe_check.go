package ccr

import (
	"fmt"

	"github.com/twitchylinux/ccr/vts"
	"gopkg.in/src-d/go-billy.v4/osfs"
)

type targetSet map[vts.Target]struct{}

// Check runs the checkers for all reachable targets against the system
// in basePath.
func (u *Universe) Check(targets []vts.TargetRef, basePath string) error {
	if !u.resolved {
		return ErrNotBuilt
	}

	var (
		evaluatedTargets = make(targetSet, 4096)
		opts             = vts.RunnerEnv{
			Dir:      basePath,
			FS:       osfs.New(basePath),
			Universe: &runtimeResolver{u, map[string]interface{}{}},
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

	for _, chkr := range u.globalCheckers {
		if err := chkr.RunCheckedTarget(nil, &opts); err != nil {
			u.logger.Error(MsgFailedCheck, err)
			return err
		}
	}
	return nil
}

func (u *Universe) checkTarget(t vts.Target, opts *vts.RunnerEnv, checked targetSet) error {
	if _, checked := checked[t]; checked {
		return nil
	}
	checked[t] = struct{}{}

	// Check dependencies first.
	if deps, hasDeps := t.(vts.DepTarget); hasDeps {
		for _, dep := range deps.Dependencies() {
			if err := u.checkTarget(dep.Target, opts, checked); err != nil {
				return vts.WrapWithTarget(err, t)
			}
		}
	}
	// Validate attributes by recursing.
	if deets, hasDetails := t.(vts.DetailedTarget); hasDetails {
		for _, attr := range deets.Attributes() {
			if err := u.checkTarget(attr.Target, opts, checked); err != nil {
				return vts.WrapWithTarget(err, t)
			}
		}
	}

	// Validate checkers defined on a class target if applicable.
	if class, hasClass := t.(vts.ClassedTarget); hasClass {
		switch n := class.Class().Target.(type) {
		case *vts.ResourceClass:
			if err := n.RunCheckers(t.(*vts.Resource), opts); err != nil {
				return u.logger.Error(MsgFailedCheck, vts.WrapWithTarget(err, t))
			}
		case *vts.AttrClass:
			if err := n.RunCheckers(t.(*vts.Attr), opts); err != nil {
				return u.logger.Error(MsgFailedCheck, vts.WrapWithTarget(err, t))
			}
		default:
			return vts.WrapWithTarget(fmt.Errorf("cannot check against class target %T", class.Class().Target), t)
		}
	}

	// Finally, validate any checkers on the target itself.
	if !t.IsClassTarget() {
		if n, hasChecks := t.(vts.CheckedTarget); hasChecks {
			for _, c := range n.Checkers() {
				ct := c.Target.(*vts.Checker)
				// Do not run global checks: they run at the end.
				if ct.Kind != vts.ChkKindGlobal {
					if err := ct.RunCheckedTarget(n, opts); err != nil {
						return u.logger.Error(MsgFailedCheck, vts.WrapWithTarget(err, t))
					}
				}
			}
		}

		// Some targets annotate a source, which can have logic for checking.
		if st, hasSrc := t.(vts.SourcedTarget); hasSrc {
			if src := st.Src(); src != nil {
				if err := u.checkAgainstSource(opts, t, src.Target); err != nil {
					return u.logger.Error(MsgFailedCheck, vts.WrapWithTarget(err, t))
				}
				if err := u.checkTarget(src.Target, opts, checked); err != nil {
					return vts.WrapWithTarget(err, src.Target)
				}
			}
		}
	}

	// As a special case, toolchain targets need to check that the binaries they
	// map exist on the system. opts.FS will point to the host system if
	// we are checking a host toolchain.
	if tc, isToolchain := t.(*vts.Toolchain); isToolchain {
		for n, p := range tc.BinaryMappings {
			if _, err := opts.FS.Stat(p); err != nil {
				return vts.WrapWithTarget(vts.WrapWithPath(fmt.Errorf("toolchain component missing: %s", n), p), tc)
			}
		}
	}
	return nil
}

func (u *Universe) checkAgainstSource(opts *vts.RunnerEnv, t vts.Target, src vts.Target) error {
	switch source := src.(type) {
	case *vts.Puesdo:
		switch source.Kind {
		case vts.FileRef, vts.DebRef:

		default:
			return fmt.Errorf("puesdo target has unsupported kind %v", source.Kind)
		}
	case *vts.Generator:
	default:
		return fmt.Errorf("cannot check against source of type %T", src)
	}
	return nil
}
