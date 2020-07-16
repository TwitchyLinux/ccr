package ccbuild

import (
	"github.com/twitchylinux/ccr/vts"
	"github.com/twitchylinux/ccr/vts/ccbuild/runners"
	"github.com/twitchylinux/ccr/vts/ccbuild/runners/syslib"
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
				"path": runners.PathCheckValid(),
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
		"file": makePuesdotarget(s, vts.FileRef),
		"deb":  makePuesdotarget(s, vts.DebRef),
	}, nil
}
