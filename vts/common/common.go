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
	"common://attrs:semver":      SemverClass,
	"common://attrs/arch:x86":    archDir["x86"],
	"common://attrs/arch:amd64":  archDir["amd64"],
	"common://attrs/arch:arm":    archDir["arm"],
	"common://attrs/arch:arm64":  archDir["arm64"],

	"common://resources:dir":                 DirResourceClass,
	"common://resources:file":                FileResourceClass,
	"common://resources:symlink":             SymlinkResourceClass,
	"common://resources:virtual":             VirtualResourceClass,
	"common://resources:binary":              BinResourceClass,
	"common://resources:binary_symlink":      BinLinkResourceClass,
	"common://resources:sys_library":         SysLibResourceClass,
	"common://resources:sys_library_symlink": SysLibLinkResourceClass,
	"common://resources:static_library":      StaticLibResourceClass,
	"common://resources:library_dir":         LibDirResourceClass,
	"common://resources:json_file":           JSONResourceClass,
	"common://resources:pkgcfg_file":         PkgcfgResourceClass,
	"common://resources:libtool_desc":        LibtoolDescResourceClass,
	"common://resources:c_header":            CHeaderResourceClass,
	"common://resources:c_headers":           CHeadersResourceClass,
	"common://resources:support_files":       SupportDirResourceClass,

	"common://resources/accounts:user":  UserResourceClass,
	"common://resources/accounts:group": GroupResourceClass,

	"common://checks:noop":                    NoopComponentChecker,
	"common://checks:file_present":            FilePresentChecker,
	"common://checks:dir_present":             DirPresentChecker,
	"common://checks:symlink_present":         SymlinkPresentChecker,
	"common://checks/formats:json_valid":      JSONResourceChecker,
	"common://checks/executable:binary":       BinaryResourceChecker,
	"common://checks:octal_string":            OctalStringChecker,
	"common://checks:boolean":                 BoolChecker,
	"common://checks:deb_info":                DebInfoChecker,
	"common://checks:semver_valid":            SemverChecker,
	"common://checks:always_fail":             DebugFailingComponentChecker,
	"common://checks:c_headers":               CHeadersChecker,
	"common://checks/filelist:present":        FilelistAllPresentChecker,
	"common://checks/universe:syslib_linking": SystemLinkChecker,

	"common://generators:dir":     DirGenerator,
	"common://generators:symlink": SymlinkGenerator,

	"common://toolchains:go":                GoToolchain,
	"common://toolchains/version:go":        GoVersion,
	"common://toolchains:gcc":               GccToolchain,
	"common://toolchains/version:gcc":       GccVersion,
	"common://toolchains:bash":              BashToolchain,
	"common://toolchains/version:bash":      BashVersion,
	"common://toolchains:make":              MakeToolchain,
	"common://toolchains/version:make":      MakeVersion,
	"common://toolchains:coreutils":         CoreutilsToolchain,
	"common://toolchains/version:coreutils": CoreutilsVersion,
	"common://toolchains:binutils":          BinutilsToolchain,
	"common://toolchains/version:binutils":  BinutilsVersion,
	"common://toolchains:diffutils":         DiffutilsToolchain,
	"common://toolchains/version:diffutils": DiffutilsVersion,
	"common://toolchains:findutils":         FindutilsToolchain,
	"common://toolchains/version:findutils": FindutilsVersion,
	"common://toolchains:patch":             PatchToolchain,
	"common://toolchains/version:patch":     PatchVersion,
	"common://toolchains:sed":               SedToolchain,
	"common://toolchains/version:sed":       SedVersion,
	"common://toolchains:grep":              GrepToolchain,
	"common://toolchains/version:grep":      GrepVersion,
	"common://toolchains:m4":                M4Toolchain,
	"common://toolchains/version:m4":        M4Version,
	"common://toolchains:automake":          AutomakeToolchain,
	"common://toolchains/version:automake":  AutomakeVersion,
	"common://toolchains:autoconf":          AutoconfToolchain,
	"common://toolchains/version:autoconf":  AutoconfVersion,
}
