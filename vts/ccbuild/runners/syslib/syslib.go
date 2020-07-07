// Package syslib implements a global check of runtime link dependencies.
package syslib

import (
	"crypto/sha256"
	"errors"
	"fmt"

	"github.com/twitchylinux/ccr/vts"
	"go.starlark.net/starlark"
)

// RuntimeLinkChecker returns a global runner that can check runtime
// link dependencies are satisfied.
func RuntimeLinkChecker() *globalChecker {
	return &globalChecker{}
}

type globalChecker struct{}

func (*globalChecker) Kind() vts.CheckerKind { return vts.ChkKindGlobal }

func (*globalChecker) String() string { return "syslib.link_checker" }

func (*globalChecker) Freeze() {}

func (*globalChecker) Truth() starlark.Bool { return true }

func (*globalChecker) Type() string { return "runner" }

func (t *globalChecker) Hash() (uint32, error) {
	h := sha256.Sum256([]byte(fmt.Sprintf("%p", t)))
	return uint32(uint32(h[0]) + uint32(h[1])<<8 + uint32(h[2])<<16 + uint32(h[3])<<24), nil
}

func (*globalChecker) Run(chkr *vts.Checker, opts *vts.RunnerEnv) error {
	dirs, err := getLibraryDirs(opts)
	if err != nil {
		return fmt.Errorf("enumerating system library dirs: %v", err)
	}
	if len(dirs) == 0 {
		return errors.New("no system libraries declared in universe")
	}

	bins, err := getBinaries(opts)
	if err != nil {
		return fmt.Errorf("enumerating binaries: %v", err)
	}
	if len(bins) == 0 {
		return errors.New("no binaries declared in universe")
	}

	return nil
}
