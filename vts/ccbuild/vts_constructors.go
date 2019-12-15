package ccbuild

import (
	"fmt"

	"github.com/twitchylinux/ccr/vts"
	"go.starlark.net/starlark"
)

func toCheckTarget(v starlark.Value) (vts.TargetRef, error) {
	if s, ok := v.(starlark.String); ok {
		return vts.TargetRef{Path: string(s)}, nil
	}
	return vts.TargetRef{}, fmt.Errorf("cannot reference check with starklark type %T (%s)", v, v.String())
}

func toDepTarget(v starlark.Value) (vts.TargetRef, error) {
	if s, ok := v.(starlark.String); ok {
		return vts.TargetRef{Path: string(s)}, nil
	}
	return vts.TargetRef{}, fmt.Errorf("cannot reference dep with starklark type %T (%s)", v, v.String())
}

func toDetailsTarget(v starlark.Value) (vts.TargetRef, error) {
	if a, ok := v.(*vts.Attr); ok {
		return vts.TargetRef{Target: a}, nil
	}
	if s, ok := v.(starlark.String); ok {
		return vts.TargetRef{Path: string(s)}, nil
	}
	return vts.TargetRef{}, fmt.Errorf("cannot reference detail with starklark type %T (%s)", v, v.String())
}

func makeAttrClass(s *Script) *starlark.Builtin {
	t := vts.TargetAttrClass

	return starlark.NewBuiltin(t.String(), func(thread *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		var name string
		var checks *starlark.List
		if err := starlark.UnpackArgs(t.String(), args, kwargs, "name", &name, "chks?", &checks); err != nil {
			return starlark.None, err
		}

		ac := &vts.AttrClass{Path: s.makePath(name), Name: name}
		if checks != nil {
			i := checks.Iterate()
			defer i.Done()
			var x starlark.Value
			for i.Next(&x) {
				v, err := toCheckTarget(x)
				if err != nil {
					return nil, fmt.Errorf("invalid check: %v", err)
				}
				ac.Checks = append(ac.Checks, v)
			}
		}

		s.targets = append(s.targets, ac)
		return starlark.None, nil
	})
}

func makeAttr(s *Script) *starlark.Builtin {
	t := vts.TargetAttr

	return starlark.NewBuiltin(t.String(), func(thread *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		var name, parent string
		var value starlark.Value
		if err := starlark.UnpackArgs(t.String(), args, kwargs, "name?", &name, "parent", &parent, "value", &value); err != nil {
			return starlark.None, err
		}

		attr := &vts.Attr{Path: s.makePath(name), Name: name, Parent: vts.TargetRef{Path: parent}, Value: value}
		// If theres no name, it must be an anonymous attr as part of another
		// target. We don't add it to the targets list.
		if name == "" {
			return attr, nil
		}
		s.targets = append(s.targets, attr)
		return starlark.None, nil
	})
}

func makeResource(s *Script) *starlark.Builtin {
	t := vts.TargetResource

	return starlark.NewBuiltin(t.String(), func(thread *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		var name, parent string
		var details, deps *starlark.List
		if err := starlark.UnpackArgs(t.String(), args, kwargs, "name", &name, "parent", &parent, "details?", &details, "deps?", &deps); err != nil {
			return starlark.None, err
		}

		r := &vts.Resource{Path: s.makePath(name), Name: name, Parent: vts.TargetRef{Path: parent}}
		if details != nil {
			i := details.Iterate()
			defer i.Done()
			var x starlark.Value
			for i.Next(&x) {
				v, err := toDetailsTarget(x)
				if err != nil {
					return nil, fmt.Errorf("invalid detail: %v", err)
				}
				r.Details = append(r.Details, v)
			}
		}
		if deps != nil {
			i := deps.Iterate()
			defer i.Done()
			var x starlark.Value
			for i.Next(&x) {
				v, err := toDepTarget(x)
				if err != nil {
					return nil, fmt.Errorf("invalid dep: %v", err)
				}
				r.Deps = append(r.Deps, v)
			}
		}

		s.targets = append(s.targets, r)
		return starlark.None, nil
	})
}

func makeResourceClass(s *Script) *starlark.Builtin {
	t := vts.TargetResourceClass

	return starlark.NewBuiltin(t.String(), func(thread *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		var name string
		var chks, deps *starlark.List
		if err := starlark.UnpackArgs(t.String(), args, kwargs, "name", &name, "chks?", &chks, "deps?", &deps); err != nil {
			return starlark.None, err
		}

		r := &vts.ResourceClass{Path: s.makePath(name), Name: name}
		if chks != nil {
			i := chks.Iterate()
			defer i.Done()
			var x starlark.Value
			for i.Next(&x) {
				v, err := toCheckTarget(x)
				if err != nil {
					return nil, fmt.Errorf("invalid check: %v", err)
				}
				r.Checks = append(r.Checks, v)
			}
		}
		if deps != nil {
			i := deps.Iterate()
			defer i.Done()
			var x starlark.Value
			for i.Next(&x) {
				v, err := toDepTarget(x)
				if err != nil {
					return nil, fmt.Errorf("invalid dep: %v", err)
				}
				r.Deps = append(r.Deps, v)
			}
		}

		s.targets = append(s.targets, r)
		return starlark.None, nil
	})
}

func makeComponent(s *Script) *starlark.Builtin {
	t := vts.TargetComponent

	return starlark.NewBuiltin(t.String(), func(thread *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		var name string
		var details, deps *starlark.List
		if err := starlark.UnpackArgs(t.String(), args, kwargs, "name", &name, "details?", &details, "deps?", &deps); err != nil {
			return starlark.None, err
		}

		r := &vts.Component{Path: s.makePath(name), Name: name}
		if details != nil {
			i := details.Iterate()
			defer i.Done()
			var x starlark.Value
			for i.Next(&x) {
				v, err := toDetailsTarget(x)
				if err != nil {
					return nil, fmt.Errorf("invalid detail: %v", err)
				}
				r.Details = append(r.Details, v)
			}
		}
		if deps != nil {
			i := deps.Iterate()
			defer i.Done()
			var x starlark.Value
			for i.Next(&x) {
				v, err := toDepTarget(x)
				if err != nil {
					return nil, fmt.Errorf("invalid dep: %v", err)
				}
				r.Deps = append(r.Deps, v)
			}
		}

		s.targets = append(s.targets, r)
		return starlark.None, nil
	})
}
