package common

import "github.com/twitchylinux/ccr/vts"

var FileResourceClass = &vts.ResourceClass{
	Path: "common://resources:file",
	Name: "file",
	Checks: []vts.TargetRef{
		{Target: FilePresentChecker},
	},
}

var DirResourceClass = &vts.ResourceClass{
	Path: "common://resources:dir",
	Name: "dir",
	Checks: []vts.TargetRef{
		{Target: DirPresentChecker},
	},
}

var BinResourceClass = &vts.ResourceClass{
	Path: "common://resources:binary",
	Name: "binary",
	Checks: []vts.TargetRef{
		{Target: BinaryResourceChecker},
	},
}

var SysLibResourceClass = &vts.ResourceClass{
	Path: "common://resources:sys_library",
	Name: "sys_library",
}

var JSONResourceClass = &vts.ResourceClass{
	Path: "common://resources:json_file",
	Name: "json_file",
	Checks: []vts.TargetRef{
		{Target: JSONResourceChecker},
	},
}

var VirtualResourceClass = &vts.ResourceClass{
	Path: "common://resources:virtual",
	Name: "virtual",
}
