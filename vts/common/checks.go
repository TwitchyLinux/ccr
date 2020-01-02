package common

import (
	"github.com/twitchylinux/ccr/vts"
	"github.com/twitchylinux/ccr/vts/ccbuild/runners"
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
