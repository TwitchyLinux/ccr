// Package syslib implements a global check of runtime link dependencies.
package syslib

import (
	"crypto/sha256"
	"debug/elf"
	"errors"
	"fmt"

	"github.com/twitchylinux/ccr/vts"
	"github.com/twitchylinux/ccr/vts/ccbuild/info"
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

func (c *globalChecker) Run(chkr *vts.Checker, opts *vts.RunnerEnv) error {
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
	for path, bin := range bins {
		if err := c.checkBinary(bin, opts); err != nil {
			return vts.WrapWithPath(vts.WrapWithTarget(err, bin), path)
		}
	}

	return nil
}

func (*globalChecker) binaryInfo(r *vts.Resource, opts *vts.RunnerEnv) (elf.FileHeader, []info.ELFSym, string, error) {
	if !r.RuntimeInfo().HasRun(info.ELFPopulator) {
		if err := info.ELFPopulator.Run(r, opts, r.RuntimeInfo()); err != nil {
			return elf.FileHeader{}, nil, "", err
		}
	}
	d, err := r.RuntimeInfo().Get(info.ELFPopulator, info.ELFHeader)
	if err != nil {
		return elf.FileHeader{}, nil, "", err
	}
	elfHeader := d.(elf.FileHeader)
	if d, err = r.RuntimeInfo().Get(info.ELFPopulator, info.ELFDynamicSymbols); err != nil {
		return elf.FileHeader{}, nil, "", err
	}
	syms := d.([]info.ELFSym)
	if d, err = r.RuntimeInfo().Get(info.ELFPopulator, info.ELFInterpreter); err != nil {
		return elf.FileHeader{}, nil, "", err
	}
	interp := d.(string)
	return elfHeader, syms, interp, nil
}

func (c *globalChecker) checkBinary(r *vts.Resource, opts *vts.RunnerEnv) error {
	_, syms, interp, err := c.binaryInfo(r, opts)
	if _, err := opts.Universe.FindByPath(interp); err != nil {
		return fmt.Errorf("couldnt read resource representing declared dynamic linker (%q): %v", interp, err)
	}

	fmt.Printf("info: %+v\n", syms)
	return err
}
