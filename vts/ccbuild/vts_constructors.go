package ccbuild

import (
	"github.com/twitchylinux/ccr/vts"
	"go.starlark.net/starlark"
)

func makeAttrClass(s *Script) *starlark.Builtin {
	t := vts.TargetAttrClass

	return starlark.NewBuiltin(t.String(), func(thread *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		var name string
		if err := starlark.UnpackArgs(t.String(), args, kwargs, "name", &name); err != nil {
			return starlark.None, err
		}
		s.targets = append(s.targets, &vts.AttrClass{Path: s.makePath(name), Name: name})
		return starlark.None, nil
	})
}

func makeAttr(s *Script) *starlark.Builtin {
	t := vts.TargetAttr

	return starlark.NewBuiltin(t.String(), func(thread *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		var name, class string
		if err := starlark.UnpackArgs(t.String(), args, kwargs, "name", &name, "parent_class", &class); err != nil {
			return starlark.None, err
		}
		s.targets = append(s.targets, &vts.Attr{Path: s.makePath(name), Name: name, ParentClass: vts.TargetRef{Path: class}})
		return starlark.None, nil
	})
}
