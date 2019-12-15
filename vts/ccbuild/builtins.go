package ccbuild

import (
	"go.starlark.net/starlark"
)

func (s *Script) makeBuiltins() (starlark.StringDict, error) {
	return starlark.StringDict{
		"attr_class":     makeAttrClass(s),
		"attr":           makeAttr(s),
		"resource":       makeResource(s),
		"resource_class": makeResourceClass(s),
		"component":      makeComponent(s),
	}, nil
}
