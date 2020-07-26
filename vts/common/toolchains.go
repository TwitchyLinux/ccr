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
			"gcc": "/bin/gcc",
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

var (
	BashToolchain = &vts.Toolchain{
		Path: "common://toolchains:bash",
		Name: "bash",
		BinaryMappings: map[string]string{
			"bash": "/bin/bash",
		},
		Details: []vts.TargetRef{
			{Target: BashVersion},
		},
	}

	BashVersion = &vts.Attr{
		Path:   "common://toolchains/version:bash",
		Name:   "bash",
		Parent: vts.TargetRef{Target: SemverClass},
		Val: &vts.ComputedValue{
			InlineScript: []byte(`
inv = run("bash", "--version")
lines = inv.output.split('\n')
if len(lines) < 2:
  broken_assumption("bash --version output format may have changed")

spl = lines[0].split(' ')
if len(spl) < 3 or spl[1] != 'bash,':
  broken_assumption("bash --version output format may have changed")
idx = spl.index('version')
if not idx:
  broken_assumption("bash --version output format may have changed")
return str(spl[idx+1]).split('(')[0]

	`),
		},
	}
)

var (
	MakeToolchain = &vts.Toolchain{
		Path: "common://toolchains:make",
		Name: "make",
		BinaryMappings: map[string]string{
			"make": "/bin/make",
		},
		Details: []vts.TargetRef{
			{Target: MakeVersion},
		},
	}

	MakeVersion = &vts.Attr{
		Path:   "common://toolchains/version:make",
		Name:   "make",
		Parent: vts.TargetRef{Target: SemverClass},
		Val: &vts.ComputedValue{
			InlineScript: []byte(`
inv = run("make", "--version")
lines = inv.output.split('\n')
if len(lines) < 2:
  broken_assumption("make --version output format may have changed")

spl = lines[0].split(' ')
if len(spl) < 3 or spl[1] != 'Make':
  broken_assumption("make --version output format may have changed")

return spl[2]
	`),
		},
	}
)
