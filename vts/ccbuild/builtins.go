package ccbuild

import (
	"errors"
	"fmt"

	"github.com/twitchylinux/ccr/vts"
	"github.com/twitchylinux/ccr/vts/ccbuild/runners"
	"github.com/twitchylinux/ccr/vts/ccbuild/runners/syslib"
	"github.com/twitchylinux/ccr/vts/common"
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
			"shell_cmd": makeBuildStep(s, vts.StepShellCmd),
		}),
		"file":   makePuesdotarget(s, vts.FileRef),
		"deb":    makePuesdotarget(s, vts.DebRef),
		"semver": mkTargetConstraint(common.SemverClass),
	}, nil
}

func makeBuildStep(s *Script, kind vts.StepKind) *starlark.Builtin {
	return starlark.NewBuiltin(string(kind), func(thread *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		var to, path, sha256, url string
		var argsOutput []string
		switch kind {
		case vts.StepUnpackGz:
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
		}

		return &vts.BuildStep{
			Kind:   kind,
			ToPath: to,
			Path:   path,
			SHA256: sha256,
			URL:    url,
			Args:   argsOutput,

			Pos: &vts.DefPosition{
				Path:  s.fPath,
				Frame: thread.CallFrame(1),
			},
		}, nil
	})
}
