package common

import "github.com/twitchylinux/ccr/vts"

var FileResourceClass = &vts.ResourceClass{
	Path: "common://resources:file",
	Name: "file",
	Checks: []vts.TargetRef{
		{Target: FilePresentChecker},
	},
}

var BinResourceClass = &vts.ResourceClass{
	Path: "common://resources:binary",
	Name: "binary",
}

var JSONResourceClass = &vts.ResourceClass{
	Path: "common://resources:json_file",
	Name: "json_file",
	Checks: []vts.TargetRef{
		{Target: JSONResourceChecker},
	},
}
