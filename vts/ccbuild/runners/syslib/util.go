package syslib

import (
	"errors"
	"fmt"

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
