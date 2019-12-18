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

var FileResourceClass = &vts.ResourceClass{
	Path: "common://resources:file",
	Name: "file",
}

var BinResourceClass = &vts.ResourceClass{
	Path: "common://resources:binary",
	Name: "binary",
}

var commonTargets = map[string]vts.Target{
	"common://attrs:path":       PathClass,
	"common://attrs:arch":       archClass,
	"common://attrs/arch:x86":   archDir["x86"],
	"common://attrs/arch:amd64": archDir["amd64"],
	"common://attrs/arch:arm":   archDir["arm"],
	"common://attrs/arch:arm64": archDir["arm64"],
	"common://resources:file":   FileResourceClass,
	"common://resources:binary": BinResourceClass,

	"common://checks:noop":                   NoopComponentChecker,
	"common://checks/formats:json_valid":     JSONResourceChecker,
	"common://checks/executable:binutil_bin": BinutilBinComponentChecker,
}
