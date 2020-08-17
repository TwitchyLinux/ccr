package common

import (
	"github.com/twitchylinux/ccr/vts"
	"github.com/twitchylinux/ccr/vts/ccbuild/runners"
)

var ArchClass = &vts.AttrClass{
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

// CheckerOptClass is the class for a string describing an option
// set on a checker which accepts options.
var CheckerOptClass = &vts.AttrClass{
	Path: "common://attrs:checker_opt",
	Name: "checker_opt",
	Checks: []vts.TargetRef{
		{Target: &vts.Checker{
			Kind:   vts.ChkKindEachAttr,
			Runner: runners.CheckerOptCheckValid(),
		}},
	},
}

// SemverClass is the class for a string describing a semantic version.
var SemverClass = &vts.AttrClass{
	Path: "common://attrs:semver",
	Name: "semver",
	Checks: []vts.TargetRef{
		{Target: SemverChecker},
	},
}

// archDir contains targets in common://attrs.
var archDir = map[string]vts.Target{
	"arch": ArchClass,
	"x86": &vts.Attr{
		Path:   "common://attrs/arch:x86",
		Name:   "x86",
		Parent: vts.TargetRef{Target: ArchClass},
	},
	"amd64": &vts.Attr{
		Path:   "common://attrs/arch:amd64",
		Name:   "amd64",
		Parent: vts.TargetRef{Target: ArchClass},
	},
	"arm": &vts.Attr{
		Path:   "common://attrs/arch:arm",
		Name:   "arm",
		Parent: vts.TargetRef{Target: ArchClass},
	},
	"arm64": &vts.Attr{
		Path:   "common://attrs/arch:arm64",
		Name:   "arm64",
		Parent: vts.TargetRef{Target: ArchClass},
	},
}

// InputLibraryClass is the class for a library which is part of a linkerscript
var InputLibraryClass = &vts.AttrClass{
	Path:       "common://attrs:ldscript_input_library",
	Name:       "ldscript_input_library",
	Repeatable: true,
	Checks: []vts.TargetRef{
		{Target: &vts.Checker{
			Kind:   vts.ChkKindEachAttr,
			Runner: runners.PathCheckValid(),
		}},
	},
}
