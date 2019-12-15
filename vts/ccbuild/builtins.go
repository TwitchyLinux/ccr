package ccbuild

import (
	"github.com/twitchylinux/ccr/vts"
	"go.starlark.net/starlark"
	"go.starlark.net/starlarkstruct"
)

func (s *Script) makeBuiltins() (starlark.StringDict, error) {
	return starlark.StringDict{
		"attr_class":     makeAttrClass(s),
		"attr":           makeAttr(s),
		"resource":       makeResource(s),
		"resource_class": makeResourceClass(s),
		"component":      makeComponent(s),
		"checker":        makeChecker(s),
		"const": starlarkstruct.FromStringDict(starlarkstruct.Default, starlark.StringDict{
			"check": starlarkstruct.FromStringDict(starlarkstruct.Default, starlark.StringDict{
				"each_resource": starlark.String(vts.ChkKindEachResource),
			}),
		}),
	}, nil
}
