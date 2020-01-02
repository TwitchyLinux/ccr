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
