package common

import "github.com/twitchylinux/ccr/vts"

var archClass = &vts.AttrClass{
	Path: "common://attrs:arch",
	Name: "arch",
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
