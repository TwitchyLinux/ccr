package common

import (
	"github.com/twitchylinux/ccr/vts"
	"github.com/twitchylinux/ccr/vts/ccbuild/runners"
)

var archClass = &vts.AttrClass{
	Path: "common://attrs:arch",
	Name: "arch",
}

// PathClass is the class for a string representing a path on the filesystem.
var PathClass = &vts.AttrClass{
	Path: "common://attrs:path",
	Name: "path",
	Checks: []vts.TargetRef{
		{Target: &vts.Checker{
			Kind:   vts.ChkKindEachAttr,
			Runner: runners.PathCheckValid(),
		}},
	},
}

// ModeClass is the class for an octal string representing the mode
// of a file.
var ModeClass = &vts.AttrClass{
	Path: "common://attrs:mode",
	Name: "mode",
	Checks: []vts.TargetRef{
		{Target: &vts.Checker{
			Kind:   vts.ChkKindEachAttr,
			Runner: runners.OctalCheckValid(),
		}},
	},
}

// TargetClass is the class for a path which is the target of a
// symlink.
var TargetClass = &vts.AttrClass{
	Path: "common://attrs:target",
	Name: "target",
	Checks: []vts.TargetRef{
		{Target: &vts.Checker{
			Kind:   vts.ChkKindEachAttr,
			Runner: runners.PathCheckValid(),
		}},
	},
}

// ModeClass is the class for a boolean.
var BoolClass = &vts.AttrClass{
	Path: "common://attrs:bool",
	Name: "bool",
	Checks: []vts.TargetRef{
		{Target: &vts.Checker{
			Kind:   vts.ChkKindEachAttr,
			Runner: runners.BooleanCheckValid(),
		}},
	},
}

var DebInfoClass = &vts.AttrClass{
	Path: "common://attrs:deb_info",
	Name: "deb_info",
	Checks: []vts.TargetRef{
		{Target: &vts.Checker{
			Kind:   vts.ChkKindEachAttr,
			Runner: runners.DebInfoCheckValid(),
		}},
	},
}

// archDir contains targets in common://attrs.
var archDir = map[string]vts.Target{
	"arch": archClass,
	"x86": &vts.Attr{
		Path:   "common://attrs/arch:x86",
		Name:   "x86",
		Parent: vts.TargetRef{Target: archClass},
	},
	"amd64": &vts.Attr{
		Path:   "common://attrs/arch:amd64",
		Name:   "amd64",
		Parent: vts.TargetRef{Target: archClass},
	},
	"arm": &vts.Attr{
		Path:   "common://attrs/arch:arm",
		Name:   "arm",
		Parent: vts.TargetRef{Target: archClass},
	},
	"arm64": &vts.Attr{
		Path:   "common://attrs/arch:arm64",
		Name:   "arm64",
		Parent: vts.TargetRef{Target: archClass},
	},
}
