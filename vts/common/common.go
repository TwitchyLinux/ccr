// Package common implements builtin vts targets.
package common

import (
	"os"
	"strings"

	"github.com/twitchylinux/ccr/vts"
)

// Resolve returns the target at the specified path.
func Resolve(path string) (vts.Target, error) {
	if !strings.HasPrefix(path, "common://") {
		return nil, os.ErrNotExist
	}

	t, ok := commonTargets[path]
	if !ok {
		return nil, os.ErrNotExist
	}
	return t, nil
}

var commonTargets = map[string]vts.Target{
	"common://attrs:path":        PathClass,
	"common://attrs:mode":        ModeClass,
	"common://attrs:bool":        BoolClass,
	"common://attrs:deb_info":    DebInfoClass,
	"common://attrs:target":      TargetClass,
	"common://attrs:checker_opt": CheckerOptClass,
	"common://attrs:arch":        ArchClass,
	"common://attrs/arch:x86":    archDir["x86"],
	"common://attrs/arch:amd64":  archDir["amd64"],
	"common://attrs/arch:arm":    archDir["arm"],
	"common://attrs/arch:arm64":  archDir["arm64"],

	"common://resources:dir":         DirResourceClass,
	"common://resources:file":        FileResourceClass,
	"common://resources:symlink":     SymlinkResourceClass,
	"common://resources:virtual":     VirtualResourceClass,
	"common://resources:binary":      BinResourceClass,
	"common://resources:sys_library": SysLibResourceClass,
	"common://resources:json_file":   JSONResourceClass,

	"common://resources/accounts:user":  UserResourceClass,
	"common://resources/accounts:group": GroupResourceClass,

	"common://checks:noop":               NoopComponentChecker,
	"common://checks:file_present":       FilePresentChecker,
	"common://checks:dir_present":        DirPresentChecker,
	"common://checks:symlink_present":    SymlinkPresentChecker,
	"common://checks/formats:json_valid": JSONResourceChecker,
	"common://checks/executable:binary":  BinaryResourceChecker,
	"common://checks:octal_string":       OctalStringChecker,
	"common://checks:boolean":            BoolChecker,
	"common://checks:deb_info":           DebInfoChecker,
	"common://checks:always_fail":        DebugFailingComponentChecker,

	"common://generators:dir":     DirGenerator,
	"common://generators:symlink": SymlinkGenerator,
}
