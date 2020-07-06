package runners

import (
	"errors"
	"fmt"
	"os"
	"strconv"

	"github.com/twitchylinux/ccr/vts"
	"go.starlark.net/starlark"
)

var errNoAttr = errors.New("no relevant attribute")

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

	return "", errNoAttr
}

func resourceMode(r *vts.Resource) (os.FileMode, error) {
	for _, attr := range r.Details {
		if attr.Target == nil {
			return 0, fmt.Errorf("unresolved target reference: %q", attr.Path)
		}
		a := attr.Target.(*vts.Attr)
		if a.Parent.Target == nil {
			return 0, fmt.Errorf("unresolved target reference: %q", a.Parent.Path)
		}
		if class := a.Parent.Target.(*vts.AttrClass); class.GlobalPath() == "common://attrs:mode" {
			if s, ok := a.Value.(starlark.String); ok {
				m, err := strconv.ParseInt(string(s), 8, 64)
				return os.FileMode(m), err
			}
			return 0, fmt.Errorf("bad type for path: want string, got %T", a.Value)
		}
	}

	// Special case: system library directories may omit the mode parameter.
	if r.Parent.Target.(*vts.ResourceClass).GlobalPath() == "common://resources:library_dir" {
		return 0755, nil
	}

	return 0, errNoAttr
}

func resourceTarget(r *vts.Resource) (string, error) {
	for _, attr := range r.Details {
		if attr.Target == nil {
			return "", fmt.Errorf("unresolved target reference: %q", attr.Path)
		}
		a := attr.Target.(*vts.Attr)
		if a.Parent.Target == nil {
			return "", fmt.Errorf("unresolved target reference: %q", a.Parent.Path)
		}
		if class := a.Parent.Target.(*vts.AttrClass); class.GlobalPath() == "common://attrs:target" {
			if s, ok := a.Value.(starlark.String); ok {
				return string(s), nil
			}
			return "", fmt.Errorf("bad type for target: want string, got %T", a.Value)
		}
	}

	return "", errNoAttr
}
