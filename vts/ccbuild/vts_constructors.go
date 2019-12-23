package ccbuild

import (
	"fmt"
	"strings"

	"github.com/twitchylinux/ccr/vts"
	"github.com/twitchylinux/ccr/vts/common"
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
	if a, ok := v.(*vts.Puesdo); ok && a.Kind == vts.FileRef {
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

func makeAttrClass(s *Script) *starlark.Builtin {
	t := vts.TargetAttrClass

	return starlark.NewBuiltin(t.String(), func(thread *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		var name string
		var checks *starlark.List
		if err := starlark.UnpackArgs(t.String(), args, kwargs, "name", &name, "chks?", &checks); err != nil {
			return starlark.None, err
		}

		ac := &vts.AttrClass{
			Path: s.makePath(name),
			Name: name,
			Pos: &vts.DefPosition{
				Path:  s.fPath,
				Frame: thread.CallFrame(1),
			},
		}
		if checks != nil {
			i := checks.Iterate()
			defer i.Done()
			var x starlark.Value
			for i.Next(&x) {
				v, err := toCheckTarget(s.path, x)
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

		parentClass := vts.TargetRef{Path: parent}
		if strings.HasPrefix(parent, ":") {
			parentClass.Path = s.path + parent
		}

		attr := &vts.Attr{
			Path:   s.makePath(name),
			Name:   name,
			Parent: parentClass,
			Value:  value,
			Pos: &vts.DefPosition{
				Path:  s.fPath,
				Frame: thread.CallFrame(1),
			},
		}
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
		var (
			name, parent, path string
			details, deps      *starlark.List
			source             starlark.Value
		)
		if err := starlark.UnpackArgs(t.String(), args, kwargs,
			// Core arguments.
			"name", &name, "parent", &parent, "details?", &details, "deps?", &deps,
			"source?", &source,
			// Helper arguments.
			"path?", &path); err != nil {
			return starlark.None, err
		}

		parentClass := vts.TargetRef{Path: parent}
		if strings.HasPrefix(parent, ":") {
			parentClass.Path = s.path + parent
		}

		r := &vts.Resource{
			Path:   s.makePath(name),
			Name:   name,
			Parent: parentClass,
			Pos: &vts.DefPosition{
				Path:  s.fPath,
				Frame: thread.CallFrame(1),
			},
		}
		if details != nil {
			i := details.Iterate()
			defer i.Done()
			var x starlark.Value
			for i.Next(&x) {
				v, err := toDetailsTarget(s.path, x)
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
				v, err := toDepTarget(s.path, x)
				if err != nil {
					return nil, fmt.Errorf("invalid dep: %v", err)
				}
				r.Deps = append(r.Deps, v)
			}
		}
		if source != nil {
			src, err := toGeneratorTarget(s.path, source)
			if err != nil {
				return nil, fmt.Errorf("invalid source: %v", err)
			}
			r.Source = &src
		}

		// Apply any helpers that were present.
		if path != "" {
			r.Details = append(r.Details, vts.TargetRef{Target: &vts.Attr{
				Parent: vts.TargetRef{Target: common.PathClass},
				Value:  starlark.String(path),
				Pos:    r.Pos,
			}})
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

		r := &vts.ResourceClass{
			Path: s.makePath(name),
			Name: name,
			Pos: &vts.DefPosition{
				Path:  s.fPath,
				Frame: thread.CallFrame(1),
			},
		}
		if chks != nil {
			i := chks.Iterate()
			defer i.Done()
			var x starlark.Value
			for i.Next(&x) {
				v, err := toCheckTarget(s.path, x)
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
				v, err := toDepTarget(s.path, x)
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
		var details, deps, checks *starlark.List
		if err := starlark.UnpackArgs(t.String(), args, kwargs, "name", &name, "details?", &details, "deps?", &deps, "chks?", &checks); err != nil {
			return starlark.None, err
		}

		r := &vts.Component{
			Path: s.makePath(name),
			Name: name,
			Pos: &vts.DefPosition{
				Path:  s.fPath,
				Frame: thread.CallFrame(1),
			},
		}
		if details != nil {
			i := details.Iterate()
			defer i.Done()
			var x starlark.Value
			for i.Next(&x) {
				v, err := toDetailsTarget(s.path, x)
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
				v, err := toDepTarget(s.path, x)
				if err != nil {
					return nil, fmt.Errorf("invalid dep: %v", err)
				}
				r.Deps = append(r.Deps, v)
			}
		}
		if checks != nil {
			i := checks.Iterate()
			defer i.Done()
			var x starlark.Value
			for i.Next(&x) {
				v, err := toCheckTarget(s.path, x)
				if err != nil {
					return nil, fmt.Errorf("invalid check: %v", err)
				}
				r.Checks = append(r.Checks, v)
			}
		}

		s.targets = append(s.targets, r)
		return starlark.None, nil
	})
}

func makeChecker(s *Script) *starlark.Builtin {
	t := vts.TargetChecker

	return starlark.NewBuiltin(t.String(), func(thread *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		var name, kind string
		var run starlark.Value
		if err := starlark.UnpackArgs(t.String(), args, kwargs, "name?", &name, "kind", &kind, "run", &run); err != nil {
			return starlark.None, err
		}

		checker := &vts.Checker{
			Path:   s.makePath(name),
			Name:   name,
			Kind:   vts.CheckerKind(kind),
			Runner: run,
			Pos: &vts.DefPosition{
				Path:  s.fPath,
				Frame: thread.CallFrame(1),
			},
		}
		// If theres no name, it must be an anonymous checker as part of another
		// target. We don't add it to the targets list.
		if name == "" {
			return checker, nil
		}
		s.targets = append(s.targets, checker)
		return starlark.None, nil
	})
}

func makeGenerator(s *Script) *starlark.Builtin {
	t := vts.TargetGenerator

	return starlark.NewBuiltin(t.String(), func(thread *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		var name string
		var run starlark.Value
		var inputs *starlark.List
		if err := starlark.UnpackArgs(t.String(), args, kwargs, "name?", &name, "inputs?", &inputs, "run?", &run); err != nil {
			return starlark.None, err
		}

		gen := &vts.Generator{
			Path:   s.makePath(name),
			Name:   name,
			Runner: run,
			Pos: &vts.DefPosition{
				Path:  s.fPath,
				Frame: thread.CallFrame(1),
			},
		}

		if inputs != nil {
			i := inputs.Iterate()
			defer i.Done()
			var x starlark.Value
			for i.Next(&x) {
				v, err := toDepTarget(s.path, x)
				if err != nil {
					return nil, fmt.Errorf("invalid input: %v", err)
				}
				gen.Inputs = append(gen.Inputs, v)
			}
		}

		// If theres no name, it must be an anonymous generator as part of
		// anohter target. We don't add it to the targets list.
		if name == "" {
			return gen, nil
		}
		s.targets = append(s.targets, gen)
		return starlark.None, nil
	})
}
