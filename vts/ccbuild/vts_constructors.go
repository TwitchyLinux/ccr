package ccbuild

import (
	"fmt"

	"github.com/twitchylinux/ccr/vts"
	"go.starlark.net/starlark"
)

func toValidatorTarget(v starlark.Value) (vts.TargetRef, error) {
	if s, ok := v.(starlark.String); ok {
		return vts.TargetRef{Path: string(s)}, nil
	}
	return vts.TargetRef{}, fmt.Errorf("cannot reference validator with starklark type %T (%s)", v, v.String())
}

func makeAttrClass(s *Script) *starlark.Builtin {
	t := vts.TargetAttrClass

	return starlark.NewBuiltin(t.String(), func(thread *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		var name string
		var validators *starlark.List
		if err := starlark.UnpackArgs(t.String(), args, kwargs, "name", &name, "chks?", &validators); err != nil {
			return starlark.None, err
		}

		ac := &vts.AttrClass{Path: s.makePath(name), Name: name}
		if validators != nil {
			i := validators.Iterate()
			defer i.Done()
			var x starlark.Value
			for i.Next(&x) {
				v, err := toValidatorTarget(x)
				if err != nil {
					return nil, fmt.Errorf("invalid validator: %v", err)
				}
				ac.Validators = append(ac.Validators, v)
			}
		}

		s.targets = append(s.targets, ac)
		return starlark.None, nil
	})
}

func makeAttr(s *Script) *starlark.Builtin {
	t := vts.TargetAttr

	return starlark.NewBuiltin(t.String(), func(thread *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		var name, class string
		var value starlark.Value
		if err := starlark.UnpackArgs(t.String(), args, kwargs, "name?", &name, "parent", &class, "value", &value); err != nil {
			return starlark.None, err
		}
		s.targets = append(s.targets, &vts.Attr{Path: s.makePath(name), Name: name, ParentClass: vts.TargetRef{Path: class}})
		return starlark.None, nil
	})
}
