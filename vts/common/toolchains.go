package common

import (
	"github.com/twitchylinux/ccr/vts"
)

var GoToolchain = &vts.Toolchain{
	Path: "common://toolchains:go",
	Name: "go",
	BinaryMappings: map[string]string{
		"go":    "/usr/local/go/bin/go",
		"gofmt": "/usr/local/go/bin/gofmt",
	},
}
