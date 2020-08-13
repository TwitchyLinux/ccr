package ccbuild

import (
	"errors"
	"fmt"
	"runtime"
	"strings"

	"github.com/twitchylinux/ccr/vts"
	"github.com/twitchylinux/ccr/vts/ccbuild/runners"
	"github.com/twitchylinux/ccr/vts/ccbuild/runners/syslib"
	"github.com/twitchylinux/ccr/vts/common"
	"github.com/twitchylinux/ccr/vts/match"
	"go.starlark.net/starlark"
	"go.starlark.net/starlarkstruct"
)

var builtinDeriveEnumRunner = starlark.NewBuiltin("valid_enum", func(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, _ []starlark.Tuple) (starlark.Value, error) {
	vals := make([]string, len(args))
	for i, a := range args {
		vals[i], _ = starlark.AsString(a)
	}
	return runners.EnumCheckValid(vals), nil
})

func mkTargetConstraint(class *vts.AttrClass) *starlark.Builtin {
	return starlark.NewBuiltin("semver", func(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, _ []starlark.Tuple) (starlark.Value, error) {
		if len(args) != 1 {
			return nil, errors.New("expected 1 argument")
		}
		return &RefComparisonConstraint{AttrClass: common.SemverClass, CompareValue: args[0]}, nil
	})
}

func mkStripPrefixOutputMapper(s *Script) *starlark.Builtin {
	return starlark.NewBuiltin("strip_prefix", func(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, _ []starlark.Tuple) (starlark.Value, error) {
		if len(args) != 1 {
			return nil, errors.New("expected 1 argument")
		}
		s, _ := starlark.AsString(args[0])
		return &match.StripPrefixOutputMapper{Prefix: strings.TrimPrefix(s, "/")}, nil
	})
}

func recommendedCPUs() int {
	n := runtime.NumCPU()
	switch {
	case n == 2:
		return 1
	case n < 5:
		return 2
	case n <= 8:
		return n - 2
	case n <= 16:
		return n - 3
	}

	return n - 4
}

func (s *Script) makeBuiltins() (starlark.StringDict, error) {
	return starlark.StringDict{
		"attr_class":     makeAttrClass(s),
		"attr":           makeAttr(s),
		"resource":       makeResource(s),
		"resource_class": makeResourceClass(s),
		"component":      makeComponent(s),
		"checker":        makeChecker(s),
		"generator":      makeGenerator(s),
		"toolchain":      makeToolchain(s),
		"build":          makeBuild(s),
		"compute":        makeComputedValue(s),
		"const": starlarkstruct.FromStringDict(starlarkstruct.Default, starlark.StringDict{
			"check": starlarkstruct.FromStringDict(starlarkstruct.Default, starlark.StringDict{
				"each_resource":  starlark.String(vts.ChkKindEachResource),
				"each_attribute": starlark.String(vts.ChkKindEachAttr),
				"each_component": starlark.String(vts.ChkKindEachComponent),
				"universe":       starlark.String(vts.ChkKindGlobal),
			}),
			"populate": starlarkstruct.FromStringDict(starlarkstruct.Default, starlark.StringDict{
				"first_file":        starlark.MakeInt(int(vts.PopulateFileFirst)),
				"matching_path":     starlark.MakeInt(int(vts.PopulateFileMatchPath)),
				"matching_filename": starlark.MakeInt(int(vts.PopulateFileMatchBasePath)),
				"all":               starlark.MakeInt(int(vts.PopulateFiles)),
			}),
		}),
		"builtin": starlarkstruct.FromStringDict(starlarkstruct.Default, starlark.StringDict{
			"json": starlarkstruct.FromStringDict(starlarkstruct.Default, starlark.StringDict{
				"check_valid": runners.JSONCheckValid(),
			}),
			"attr": starlarkstruct.FromStringDict(starlarkstruct.Default, starlark.StringDict{
				"path":   runners.PathCheckValid(),
				"semver": runners.SemverCheckValid(),
			}),
			"debug": starlarkstruct.FromStringDict(starlarkstruct.Default, starlark.StringDict{
				"generator_input": runners.GenerateDebugManifest(),
			}),
			"syslib": starlarkstruct.FromStringDict(starlarkstruct.Default, starlark.StringDict{
				"link_checker": syslib.RuntimeLinkChecker(),
			}),
		}),
		"derive": starlarkstruct.FromStringDict(starlarkstruct.Default, starlark.StringDict{
			"check": starlarkstruct.FromStringDict(starlarkstruct.Default, starlark.StringDict{
				"valid_enum": builtinDeriveEnumRunner,
			}),
		}),
		"step": starlarkstruct.FromStringDict(starlarkstruct.Default, starlark.StringDict{
			"unpack_gz": makeBuildStep(s, vts.StepUnpackGz),
			"unpack_xz": makeBuildStep(s, vts.StepUnpackXz),
			"shell_cmd": makeBuildStep(s, vts.StepShellCmd),
			"configure": makeBuildStep(s, vts.StepConfigure),
			"patch":     makeBuildStep(s, vts.StepPatch),
		}),
		"host": starlarkstruct.FromStringDict(starlarkstruct.Default, starlark.StringDict{
			"recommended_cpus": starlark.MakeInt(recommendedCPUs()),
		}),
		"strip_prefix": mkStripPrefixOutputMapper(s),
		"file":         makePuesdotarget(s, vts.FileRef),
		"deb":          makePuesdotarget(s, vts.DebRef),
		"sieve":        makeSieve(s),
		"sieve_prefix": makeSievePrefix(s),
		"semver":       mkTargetConstraint(common.SemverClass),
	}, nil
}

func makeBuildStep(s *Script, kind vts.StepKind) *starlark.Builtin {
	return starlark.NewBuiltin(string(kind), func(thread *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		var (
			to, path, sha256, url string
			dir                   string
			argsOutput            []string
			argDict               map[string]string
			patchLevel            int
		)
		switch kind {
		case vts.StepUnpackGz, vts.StepUnpackXz:
			if err := starlark.UnpackArgs(string(kind), args, kwargs,
				"to?", &to, "path?", &path, "sha256?", &sha256, "url?", &url); err != nil {
				return starlark.None, err
			}
		case vts.StepShellCmd:
			argsOutput = make([]string, len(args))
			for i, a := range args {
				s, ok := a.(starlark.String)
				if !ok {
					return starlark.None, fmt.Errorf("arg %d was %T, want string", i, a)
				}
				argsOutput[i] = string(s)
			}
		case vts.StepConfigure:
			var sArgs, sVars *starlark.Dict
			argDict = map[string]string{}
			if err := starlark.UnpackArgs(string(kind), args, kwargs,
				"path?", &path, "dir?", &dir, "args?", &sArgs, "vars?", &sVars); err != nil {
				return starlark.None, err
			}

			if sArgs != nil {
				for _, name := range sArgs.Keys() {
					n, ok := name.(starlark.String)
					if !ok {
						return nil, fmt.Errorf("invalid args key: %v", name)
					}
					v, _, err := sArgs.Get(name)
					if err != nil {
						return starlark.None, err
					}
					v2, ok := v.(starlark.String)
					if !ok {
						return nil, fmt.Errorf("invalid args value: %v", v)
					}
					argDict[string(n)] = string(v2)
				}
			}
			if sVars != nil {
				for _, name := range sVars.Keys() {
					n, ok := name.(starlark.String)
					if !ok {
						return nil, fmt.Errorf("invalid vars key: %v", name)
					}
					v, _, err := sVars.Get(name)
					if err != nil {
						return starlark.None, err
					}
					v2, ok := v.(starlark.String)
					if !ok {
						return nil, fmt.Errorf("invalid vars value: %v", v)
					}
					argsOutput = append(argsOutput, fmt.Sprintf("%s=%s", string(n), string(v2)))
				}
			}

		case vts.StepPatch:
			if err := starlark.UnpackArgs(string(kind), args, kwargs, "path?", &path, "to?", &to, "strip_prefixes?", &patchLevel); err != nil {
				return starlark.None, err
			}
		}

		return &vts.BuildStep{
			Kind:       kind,
			Dir:        dir,
			ToPath:     to,
			Path:       path,
			SHA256:     sha256,
			URL:        url,
			Args:       argsOutput,
			NamedArgs:  argDict,
			PatchLevel: patchLevel,

			Pos: &vts.DefPosition{
				Path:  s.fPath,
				Frame: thread.CallFrame(1),
			},
		}, nil
	})
}
