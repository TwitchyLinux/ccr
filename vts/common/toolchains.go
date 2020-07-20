package common

import (
	"github.com/twitchylinux/ccr/vts"
)

var (
	GoToolchain = &vts.Toolchain{
		Path: "common://toolchains:go",
		Name: "go",
		BinaryMappings: map[string]string{
			"go":    "/usr/local/go/bin/go",
			"gofmt": "/usr/local/go/bin/gofmt",
		},
		Details: []vts.TargetRef{
			{Target: GoVersion},
		},
	}

	GoVersion = &vts.Attr{
		Path:   "common://toolchains/version:go",
		Name:   "go",
		Parent: vts.TargetRef{Target: SemverClass},
		Val: &vts.ComputedValue{
			ReadWrite: true,
			InlineScript: []byte(`
inv = run("go", "version")
spl = inv.output.split(' ')
if len(spl) < 4 or not spl[2].startswith("go") or spl[2].count(".") < 2:
  broken_assumption("go version output format may have changed")
return spl[2][2:]
`),
		},
	}
)

var (
	GccToolchain = &vts.Toolchain{
		Path: "common://toolchains:gcc",
		Name: "gcc",
		BinaryMappings: map[string]string{
			"gcc": "/usr/bin/gcc",
		},
		Details: []vts.TargetRef{
			{Target: GccVersion},
		},
	}

	GccVersion = &vts.Attr{
		Path:   "common://toolchains/version:gcc",
		Name:   "gcc",
		Parent: vts.TargetRef{Target: SemverClass},
		Val: &vts.ComputedValue{
			InlineScript: []byte(`
inv = run("gcc", "--version")
lines = inv.output.split('\n')
if len(lines) < 2:
  broken_assumption("gcc --version output format may have changed")

spl = lines[0].split(' ')
if len(spl) < 3 or spl[0] != 'gcc':
  broken_assumption("gcc --version output format may have changed")

return spl[-1]
	`),
		},
	}
)
