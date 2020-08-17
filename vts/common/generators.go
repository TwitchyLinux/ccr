package common

import (
	"github.com/twitchylinux/ccr/vts"
	"github.com/twitchylinux/ccr/vts/ccbuild/runners"
)

var DirGenerator = &vts.Generator{
	Path:   "common://generators:dir",
	Name:   "dir",
	Runner: runners.GenerateDir(),
}

var SymlinkGenerator = &vts.Generator{
	Path:   "common://generators:symlink",
	Name:   "symlink",
	Runner: runners.GenerateSymlink(),
}

var SysLibUnionLinkerscript = &vts.Generator{
	Path:   "common://generators:syslib_union_linkerscript",
	Name:   "syslib_union_linkerscript",
	Runner: runners.GenerateUnionLinkerscript(),
}
