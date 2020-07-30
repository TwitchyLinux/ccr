package ccbuild

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/twitchylinux/ccr/vts"
	"go.starlark.net/starlark"
)

func makeToolchain(s *Script) *starlark.Builtin {
	t := vts.TargetToolchain

	return starlark.NewBuiltin(t.String(), func(thread *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		var name string
		var deps, details *starlark.List
		var binaries *starlark.Dict
		if err := starlark.UnpackArgs(t.String(), args, kwargs, "name", &name, "deps?", &deps, "details?", &details, "binaries?", &binaries); err != nil {
			return starlark.None, err
		}

		tc := &vts.Toolchain{
			Path:           s.makePath(name),
			Name:           name,
			BinaryMappings: map[string]string{},
			Pos: &vts.DefPosition{
				Path:  s.fPath,
				Frame: thread.CallFrame(1),
			},
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
				tc.Deps = append(tc.Deps, v)
			}
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
				tc.Details = append(tc.Details, v)
			}
		}
		if binaries != nil {
			for _, name := range binaries.Keys() {
				n, ok := name.(starlark.String)
				if !ok {
					return nil, fmt.Errorf("invalid binary key: %v", name)
				}
				v, _, err := binaries.Get(name)
				if err != nil {
					return starlark.None, nil
				}
				v2, ok := v.(starlark.String)
				if !ok {
					return nil, fmt.Errorf("invalid binary value: %v", v)
				}
				tc.BinaryMappings[string(n)] = string(v2)
			}
		}

		s.targets = append(s.targets, tc)
		return starlark.None, nil
	})
}

func makeSieve(s *Script) *starlark.Builtin {
	t := vts.TargetSieve

	return starlark.NewBuiltin(t.String(), func(thread *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		var name string
		var inputs, exGlobs *starlark.List
		if err := starlark.UnpackArgs(t.String(), args, kwargs, "name?", &name, "inputs?", &inputs,
			"exclude?", &exGlobs); err != nil {
			return starlark.None, err
		}

		st := &vts.Sieve{
			Name:         name,
			TargetPath:   s.makePath(name),
			ContractPath: s.fPath,
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
				st.Inputs = append(st.Inputs, v)
			}
		}
		if exGlobs != nil {
			i := exGlobs.Iterate()
			defer i.Done()
			var x starlark.Value
			for i.Next(&x) {
				v, ok := x.(starlark.String)
				if !ok {
					return nil, fmt.Errorf("invalid exclude pattern: %T", x)
				}
				st.ExcludeGlobs = append(st.ExcludeGlobs, string(v))
			}
		}

		// If theres no name, it must be an anonymous target as part of another
		// target. We don't add it to the targets list.
		if name == "" {
			return st, nil
		}
		s.targets = append(s.targets, st)
		return starlark.None, nil
	})
}

func makeComputedValue(s *Script) *starlark.Builtin {
	return starlark.NewBuiltin("compute", func(thread *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		var fname, fun, code string

		if len(kwargs) == 0 && len(args) == 1 {
			c, ok := args[0].(starlark.String)
			if !ok {
				return starlark.None, fmt.Errorf("compute in unary-argument form must be called with a string, got %T", args[0])
			}
			code = string(c)
		} else {
			if err := starlark.UnpackArgs("compute", args, kwargs, "path?", &fname, "run?", &fun, "code?", &code); err != nil {
				return starlark.None, err
			}
			fname = filepath.Join(filepath.Dir(s.fPath), fname)
		}

		wd, _ := os.Getwd()
		return &vts.ComputedValue{
			ContractDir:  filepath.Join(wd, filepath.Dir(s.fPath)),
			ContractPath: s.fPath,
			Filename:     fname,
			Func:         fun,
			InlineScript: []byte(code),
			Pos: &vts.DefPosition{
				Path:  s.fPath,
				Frame: thread.CallFrame(1),
			},
		}, nil
	})
}

func makePuesdotarget(s *Script, kind vts.PuesdoKind) *starlark.Builtin {
	t := vts.TargetPuesdo

	return starlark.NewBuiltin(t.String(), func(thread *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		var path, sha256, url string
		var name string
		var host bool
		var details *starlark.List
		if err := starlark.UnpackArgs(t.String(), args, kwargs, "path?", &path,
			"name?", &name, "details?", &details, "host?", &host,
			"sha256?", &sha256, "url?", &url); err != nil {
			return starlark.None, err
		}

		pt := &vts.Puesdo{
			Kind:         kind,
			TargetPath:   s.makePath(name),
			Name:         name,
			Host:         host,
			ContractPath: s.fPath,
			Pos: &vts.DefPosition{
				Path:  s.fPath,
				Frame: thread.CallFrame(1),
			},

			Path:   path,
			SHA256: sha256,
			URL:    url,
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
				pt.Details = append(pt.Details, v)
			}
		}

		// If theres no name, it must be an anonymous target as part of another
		// target. We don't add it to the targets list.
		if name == "" {
			return pt, nil
		}
		s.targets = append(s.targets, pt)
		return starlark.None, nil
	})
}
