package common

import (
	"github.com/twitchylinux/ccr/vts"
	"github.com/twitchylinux/ccr/vts/ccbuild/runners"
	"github.com/twitchylinux/ccr/vts/ccbuild/runners/syslib"
)

var JSONResourceChecker = &vts.Checker{
	Path:   "common://checks/formats:json_valid",
	Name:   "json_valid",
	Kind:   vts.ChkKindEachResource,
	Runner: runners.JSONCheckValid(),
}

var BinaryResourceChecker = &vts.Checker{
	Path:   "common://checks/executable:binary",
	Name:   "binary",
	Kind:   vts.ChkKindEachResource,
	Runner: runners.BinaryCheckValid(),
}

var ScriptResourceChecker = &vts.Checker{
	Path:   "common://checks/executable:script",
	Name:   "script",
	Kind:   vts.ChkKindEachResource,
	Runner: runners.ScriptCheckValid(),
}

var FilePresentChecker = &vts.Checker{
	Path:   "common://checks:file_present",
	Name:   "file_present",
	Kind:   vts.ChkKindEachResource,
	Runner: runners.FileCheckPresent(),
}

var DirPresentChecker = &vts.Checker{
	Path:   "common://checks:dir_present",
	Name:   "dir_present",
	Kind:   vts.ChkKindEachResource,
	Runner: runners.DirCheckPresent(),
}

var SymlinkPresentChecker = &vts.Checker{
	Path:   "common://checks:symlink_present",
	Name:   "symlink_present",
	Kind:   vts.ChkKindEachResource,
	Runner: runners.SymlinkCheckPresent(),
}

var NoopComponentChecker = &vts.Checker{
	Path:   "common://checks:noop",
	Name:   "noop",
	Kind:   vts.ChkKindEachComponent,
	Runner: runners.NoopCheckComponent(),
}

var DebugFailingComponentChecker = &vts.Checker{
	Path:   "common://checks:always_fail",
	Name:   "always_fail",
	Kind:   vts.ChkKindEachComponent,
	Runner: runners.FailingComponentChecker(),
}

var OctalStringChecker = &vts.Checker{
	Path:   "common://checks:octal_string",
	Name:   "octal_string",
	Kind:   vts.ChkKindEachAttr,
	Runner: runners.OctalCheckValid(),
}

var BoolChecker = &vts.Checker{
	Path:   "common://checks:boolean",
	Name:   "boolean",
	Kind:   vts.ChkKindEachAttr,
	Runner: runners.BooleanCheckValid(),
}

var DebInfoChecker = &vts.Checker{
	Path:   "common://checks:deb_info",
	Name:   "deb_info",
	Kind:   vts.ChkKindEachAttr,
	Runner: runners.DebInfoCheckValid(),
}

var SemverChecker = &vts.Checker{
	Path:   "common://checks:semver_valid",
	Name:   "semver_valid",
	Kind:   vts.ChkKindEachAttr,
	Runner: runners.SemverCheckValid(),
}

var FilelistAllPresentChecker = &vts.Checker{
	Path:   "common://checks/filelist:present",
	Name:   "present",
	Kind:   vts.ChkKindEachResource,
	Runner: runners.FilelistCheckAllFiles(0, false),
}

var CHeadersChecker = &vts.Checker{
	Path:   "common://checks:c_headers",
	Name:   "c_headers",
	Kind:   vts.ChkKindEachResource,
	Runner: runners.DirOnlyCxxHeaders(),
}

var SystemLinkChecker = &vts.Checker{
	Path:   "common://checks/universe:syslib_linking",
	Name:   "syslib_linking",
	Kind:   vts.ChkKindGlobal,
	Runner: syslib.RuntimeLinkChecker(),
}
