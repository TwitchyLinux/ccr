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

var BinutilBinComponentChecker = &vts.Checker{
	Path:   "common://checks/executable:binutil_bin",
	Name:   "binutil_bin",
	Kind:   vts.ChkKindEachComponent,
	Runner: runners.BinutilCheckComponent(),
}

var NoopComponentChecker = &vts.Checker{
	Path:   "common://checks:noop",
	Name:   "noop",
	Kind:   vts.ChkKindEachComponent,
	Runner: runners.NoopCheckComponent(),
}
