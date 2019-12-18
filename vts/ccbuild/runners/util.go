package runners

import (
	"errors"
	"fmt"

	"github.com/twitchylinux/ccr/vts"
	"go.starlark.net/starlark"
)

func resourcePath(r *vts.Resource) (string, error) {
	for _, attr := range r.Details {
		if attr.Target == nil {
			return "", fmt.Errorf("unresolved target reference: %q", attr.Path)
		}
		a := attr.Target.(*vts.Attr)
		if a.Parent.Target == nil {
			return "", fmt.Errorf("unresolved target reference: %q", a.Parent.Path)
		}
		if class := a.Parent.Target.(*vts.AttrClass); class.GlobalPath() == "common://attrs:path" {
			if s, ok := a.Value.(starlark.String); ok {
				return string(s), nil
			}
			return "", fmt.Errorf("bad type for path: want string, got %T", a.Value)
		}
	}

	return "", errors.New("no path specified")
}
