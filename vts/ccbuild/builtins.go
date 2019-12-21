package ccbuild

import (
	"github.com/twitchylinux/ccr/vts"
	"github.com/twitchylinux/ccr/vts/ccbuild/runners"
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
		"const": starlarkstruct.FromStringDict(starlarkstruct.Default, starlark.StringDict{
			"check": starlarkstruct.FromStringDict(starlarkstruct.Default, starlark.StringDict{
				"each_resource":  starlark.String(vts.ChkKindEachResource),
				"each_attribute": starlark.String(vts.ChkKindEachAttr),
				"each_component": starlark.String(vts.ChkKindEachComponent),
			}),
		}),
		"builtin": starlarkstruct.FromStringDict(starlarkstruct.Default, starlark.StringDict{
			"json": starlarkstruct.FromStringDict(starlarkstruct.Default, starlark.StringDict{
				"check_valid": runners.JSONCheckValid(),
			}),
			"attr": starlarkstruct.FromStringDict(starlarkstruct.Default, starlark.StringDict{
				"path": runners.PathCheckValid(),
			}),
		}),
		"derive": starlarkstruct.FromStringDict(starlarkstruct.Default, starlark.StringDict{
			"check": starlarkstruct.FromStringDict(starlarkstruct.Default, starlark.StringDict{
				"valid_enum": builtinDeriveEnumRunner,
			}),
		}),
		"file": makePuesdotarget(s, vts.FileRef),
	}, nil
}
