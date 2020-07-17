package runners

import (
	"errors"
	"fmt"
	"os"
	"strconv"

	"github.com/twitchylinux/ccr/proc"
	"github.com/twitchylinux/ccr/vts"
	"go.starlark.net/starlark"
)

var errNoAttr = errors.New("no relevant attribute")

func resourcePath(r *vts.Resource, env *vts.RunnerEnv) (string, error) {
	for _, attr := range r.Details {
		if attr.Target == nil {
			return "", fmt.Errorf("unresolved target reference: %q", attr.Path)
		}
		a := attr.Target.(*vts.Attr)
		if a.Parent.Target == nil {
			return "", fmt.Errorf("unresolved target reference: %q", a.Parent.Path)
		}
		if class := a.Parent.Target.(*vts.AttrClass); class.GlobalPath() == "common://attrs:path" {
			v, err := a.Value(r, env, proc.EvalComputedAttribute)
			if err != nil {
				return "", err
			}
			if s, ok := v.(starlark.String); ok {
				return string(s), nil
			}
			return "", fmt.Errorf("bad type for path: want string, got %T", v)
		}
	}

	return "", errNoAttr
}

func resourceMode(r *vts.Resource, env *vts.RunnerEnv) (os.FileMode, error) {
	for _, attr := range r.Details {
		if attr.Target == nil {
			return 0, fmt.Errorf("unresolved target reference: %q", attr.Path)
		}
		a := attr.Target.(*vts.Attr)
		if a.Parent.Target == nil {
			return 0, fmt.Errorf("unresolved target reference: %q", a.Parent.Path)
		}
		if class := a.Parent.Target.(*vts.AttrClass); class.GlobalPath() == "common://attrs:mode" {
			v, err := a.Value(r, env, proc.EvalComputedAttribute)
			if err != nil {
				return 0, err
			}
			if s, ok := v.(starlark.String); ok {
				m, err := strconv.ParseInt(string(s), 8, 64)
				return os.FileMode(m), err
			}
			return 0, fmt.Errorf("bad type for path: want string, got %T", v)
		}
	}

	// Special case: system library directories may omit the mode parameter.
	// TODO: Unspecial-case this, perhaps with a new default_mode attribute / attribute-class?
	if r.Parent.Target.(*vts.ResourceClass).GlobalPath() == "common://resources:library_dir" {
		return 0755, nil
	}

	return 0, errNoAttr
}

func resourceTarget(r *vts.Resource, env *vts.RunnerEnv) (string, error) {
	for _, attr := range r.Details {
		if attr.Target == nil {
			return "", fmt.Errorf("unresolved target reference: %q", attr.Path)
		}
		a := attr.Target.(*vts.Attr)
		if a.Parent.Target == nil {
			return "", fmt.Errorf("unresolved target reference: %q", a.Parent.Path)
		}
		if class := a.Parent.Target.(*vts.AttrClass); class.GlobalPath() == "common://attrs:target" {
			v, err := a.Value(r, env, proc.EvalComputedAttribute)
			if err != nil {
				return "", err
			}
			if s, ok := v.(starlark.String); ok {
				return string(s), nil
			}
			return "", fmt.Errorf("bad type for target: want string, got %T", v)
		}
	}

	return "", errNoAttr
}
