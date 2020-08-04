package common

import (
	"github.com/twitchylinux/ccr/vts"
)

var FileResourceClass = &vts.ResourceClass{
	Path: "common://resources:file",
	Name: "file",
	Checks: []vts.TargetRef{
		{Target: FilePresentChecker},
	},
	PopStrategy: vts.PopulateFileMatchPath,
}

var DirResourceClass = &vts.ResourceClass{
	Path: "common://resources:dir",
	Name: "dir",
	Checks: []vts.TargetRef{
		{Target: DirPresentChecker},
	},
}
var LibDirResourceClass = &vts.ResourceClass{
	Path: "common://resources:library_dir",
	Name: "library_dir",
	Checks: []vts.TargetRef{
		{Target: DirPresentChecker},
	},
}

var SymlinkResourceClass = &vts.ResourceClass{
	Path: "common://resources:symlink",
	Name: "symlink",
	Checks: []vts.TargetRef{
		{Target: SymlinkPresentChecker},
	},
}
var BinResourceClass = &vts.ResourceClass{
	Path: "common://resources:binary",
	Name: "binary",
	Checks: []vts.TargetRef{
		{Target: BinaryResourceChecker},
	},
	PopStrategy: vts.PopulateFileMatchPath,
}

var SysLibResourceClass = &vts.ResourceClass{
	Path:        "common://resources:sys_library",
	Name:        "sys_library",
	PopStrategy: vts.PopulateFileMatchPath,
}
var SysLibLinkResourceClass = &vts.ResourceClass{
	Path: "common://resources:sys_library_symlink",
	Name: "sys_library_symlink",
	Checks: []vts.TargetRef{
		{Target: SymlinkPresentChecker},
	},
	PopStrategy: vts.PopulateFileMatchPath,
}
var StaticLibResourceClass = &vts.ResourceClass{
	Path: "common://resources:static_library",
	Name: "static_library",
	Checks: []vts.TargetRef{
		{Target: FilePresentChecker},
	},
	PopStrategy: vts.PopulateFileMatchPath,
}

var JSONResourceClass = &vts.ResourceClass{
	Path: "common://resources:json_file",
	Name: "json_file",
	Checks: []vts.TargetRef{
		{Target: JSONResourceChecker},
	},
	PopStrategy: vts.PopulateFileMatchPath,
}
var PkgcfgResourceClass = &vts.ResourceClass{
	Path: "common://resources:pkgcfg_file",
	Name: "pkgcfg_file",
	Checks: []vts.TargetRef{
		{Target: FilePresentChecker},
	},
	PopStrategy: vts.PopulateFileMatchPath,
}
var LibtoolDescResourceClass = &vts.ResourceClass{
	Path: "common://resources:libtool_desc",
	Name: "libtool_desc",
	Checks: []vts.TargetRef{
		{Target: FilePresentChecker},
	},
	PopStrategy: vts.PopulateFileMatchPath,
}

var VirtualResourceClass = &vts.ResourceClass{
	Path: "common://resources:virtual",
	Name: "virtual",
}

var CHeadersResourceClass = &vts.ResourceClass{
	Path: "common://resources:c_headers",
	Name: "c_headers",
	Checks: []vts.TargetRef{
		{Target: CHeadersChecker},
	},
	PopStrategy: vts.PopulateFiles,
}
var CHeaderResourceClass = &vts.ResourceClass{
	Path: "common://resources:c_header",
	Name: "c_header",
	Checks: []vts.TargetRef{
		{Target: FilePresentChecker},
	},
	PopStrategy: vts.PopulateFileMatchPath,
}
