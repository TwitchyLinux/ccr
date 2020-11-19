package ccbuild

import (
	"fmt"
	"strings"

	"github.com/twitchylinux/ccr/vts"
	"go.starlark.net/starlark"
)

func toCheckTarget(currentPath string, v starlark.Value) (vts.TargetRef, error) {
	if a, ok := v.(*vts.Checker); ok {
		return vts.TargetRef{Target: a}, nil
	}
	if s, ok := v.(starlark.String); ok {
		if strings.HasPrefix(string(s), ":") {
			return vts.TargetRef{Path: currentPath + string(s)}, nil
		}
		return vts.TargetRef{Path: string(s)}, nil
	}
	return vts.TargetRef{}, fmt.Errorf("cannot reference check with starklark type %T (%s)", v, v.String())
}

func toDepTarget(currentPath string, v starlark.Value) (vts.TargetRef, error) {
	if s, ok := v.(starlark.String); ok {
		if strings.HasPrefix(string(s), ":") {
			return vts.TargetRef{Path: currentPath + string(s)}, nil
		}
		return vts.TargetRef{Path: string(s)}, nil
	}
	if constraint, isConstraint := v.(*RefComparisonConstraint); isConstraint {
		base, err := toDepTarget(currentPath, constraint.Target)
		if err != nil {
			return vts.TargetRef{}, fmt.Errorf("constraint target: %v", err)
		}
		base.Constraints = append(base.Constraints, vts.RefConstraint{
			Meta:   vts.TargetRef{Target: constraint.AttrClass},
			Params: []starlark.Value{starlark.String(constraint.Op.String()), constraint.CompareValue},
			Eval:   constraint,
		})
		return base, nil
	}
	return vts.TargetRef{}, fmt.Errorf("cannot reference dep with starklark type %T (%s)", v, v.String())
}

func toDetailsTarget(currentPath string, v starlark.Value) (vts.TargetRef, error) {
	if a, ok := v.(*vts.Attr); ok {
		return vts.TargetRef{Target: a}, nil
	}
	if s, ok := v.(starlark.String); ok {
		if strings.HasPrefix(string(s), ":") {
			return vts.TargetRef{Path: currentPath + string(s)}, nil
		}
		return vts.TargetRef{Path: string(s)}, nil
	}
	return vts.TargetRef{}, fmt.Errorf("cannot reference detail with starklark type %T (%s)", v, v.String())
}

func toGeneratorTarget(currentPath string, v starlark.Value) (vts.TargetRef, error) {
	if a, ok := v.(*vts.Generator); ok {
		return vts.TargetRef{Target: a}, nil
	}
	if a, ok := v.(*vts.Build); ok {
		return vts.TargetRef{Target: a}, nil
	}
	if a, ok := v.(*vts.Sieve); ok {
		return vts.TargetRef{Target: a}, nil
	}
	if a, ok := v.(*vts.Puesdo); ok && (a.Kind == vts.FileRef || a.Kind == vts.DebRef) {
		return vts.TargetRef{Target: a}, nil
	}
	if s, ok := v.(starlark.String); ok {
		if strings.HasPrefix(string(s), ":") {
			return vts.TargetRef{Path: currentPath + string(s)}, nil
		}
		return vts.TargetRef{Path: string(s)}, nil
	}
	return vts.TargetRef{}, fmt.Errorf("cannot reference generator with starklark type %T (%s)", v, v.String())
}

func toBuildTarget(currentPath string, v starlark.Value) (vts.TargetRef, error) {
	if a, ok := v.(*vts.Build); ok {
		return vts.TargetRef{Target: a}, nil
	}
	if s, ok := v.(starlark.String); ok {
		if strings.HasPrefix(string(s), ":") {
			return vts.TargetRef{Path: currentPath + string(s)}, nil
		}
		return vts.TargetRef{Path: string(s)}, nil
	}
	return vts.TargetRef{}, fmt.Errorf("cannot reference generator with starklark type %T (%s)", v, v.String())
}
