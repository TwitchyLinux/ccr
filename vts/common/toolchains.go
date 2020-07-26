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

var (
	CoreutilsToolchain = &vts.Toolchain{
		Path: "common://toolchains:coreutils",
		Name: "coreutils",
		BinaryMappings: map[string]string{
			"echo":  "/bin/echo",
			"env":   "/bin/env",
			"false": "/bin/false",
			"true":  "/bin/true",
			"sleep": "/bin/sleep",
			"pwd":   "/bin/pwd",

			"chgrp":    "/bin/chgrp",
			"chown":    "/bin/chown",
			"chmod":    "/bin/chmod",
			"cp":       "/bin/cp",
			"dd":       "/bin/dd",
			"df":       "/bin/df",
			"install":  "/bin/install",
			"ln":       "/bin/ln",
			"ls":       "/bin/ls",
			"cat":      "/bin/cat",
			"readlink": "/bin/readlink",
			"stat":     "/bin/stat",
			"dir":      "/bin/dir",
			// "dircolors":   "/bin/dircolors",
			"mkdir":    "/bin/mkdir",
			"mkfifo":   "/bin/mkfifo",
			"mknod":    "/bin/mknod",
			"mktemp":   "/bin/mktemp",
			"mv":       "/bin/mv",
			"realpath": "/bin/realpath",
			"rm":       "/bin/rm",
			"rmdir":    "/bin/rmdir",
			"shred":    "/bin/shred",
			"sync":     "/bin/sync",
			"touch":    "/bin/touch",
			"truncate": "/bin/truncate",
			"vdir":     "/bin/vdir",

			// "b2sum":     "/bin/b2sum",
			"base32":    "/bin/base32",
			"base64":    "/bin/base64",
			"cksum":     "/bin/cksum",
			"comm":      "/bin/comm",
			"csplit":    "/bin/csplit",
			"cut":       "/bin/cut",
			"expand":    "/bin/expand",
			"unexpand":  "/bin/unexpand",
			"fold":      "/bin/fold",
			"md5sum":    "/bin/md5sum",
			"sha1sum":   "/bin/sha1sum",
			"sha256sum": "/bin/sha256sum",
			"sha512sum": "/bin/sha512sum",
			"sort":      "/bin/sort",
			"split":     "/bin/split",
			"sum":       "/bin/sum",
			"tail":      "/bin/tail",
			"head":      "/bin/head",
			"tr":        "/bin/tr",
			"wc":        "/bin/wc",
			"uniq":      "/bin/uniq",
			"arch":      "/bin/arch",
			"basename":  "/bin/basename",
			"chroot":    "/sbin/chroot",
			"date":      "/bin/date",
			"dirname":   "/bin/dirname",
			"du":        "/bin/du",
			"logname":   "/bin/logname",
			"nice":      "/bin/nice",
			"nohup":     "/bin/nohup",
			"pathchk":   "/bin/pathchk",
			"printenv":  "/bin/printenv",
			"printf":    "/bin/printf",
			"stdbuf":    "/bin/stdbuf",
			"stty":      "/bin/stty",
			"tee":       "/bin/tee",
			"test":      "/bin/test",
			"timeout":   "/bin/timeout",
			"tty":       "/bin/tty",
			"uname":     "/bin/uname",
			"unlink":    "/bin/unlink",
			"uptime":    "/bin/uptime",
			"users":     "/bin/users",
			"who":       "/bin/who",
			"whoami":    "/bin/whoami",
			"yes":       "/bin/yes",
			"[":         "/bin/[",
		},
		Details: []vts.TargetRef{
			{Target: CoreutilsVersion},
		},
	}

	CoreutilsVersion = &vts.Attr{
		Path:   "common://toolchains/version:coreutils",
		Name:   "coreutils",
		Parent: vts.TargetRef{Target: SemverClass},
		Val: &vts.ComputedValue{
			InlineScript: []byte(`
inv = run("chown", "--version")
lines = inv.output.split('\n')
if len(lines) < 2:
  broken_assumption("chown --version output format may have changed")

if not lines[0].startswith('chown (GNU coreutils) '):
  broken_assumption("chown --version output format may have changed")

spl = lines[0].split(' ')
if len(spl) < 3:
  broken_assumption("chown --version output format may have changed")
return spl[len(spl)-1]
	`),
		},
	}
)
