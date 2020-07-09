package info

import (
	"debug/elf"
	"errors"
	"fmt"
	"io/ioutil"

	"github.com/twitchylinux/ccr/vts"
)

// ELF populator data keys.
const (
	ELFHeader         = "elf-header"
	ELFDynamicSymbols = "elf-dynamic-symbols"
	ELFInterpreter    = "elf-interp"
	ELFDeps           = "elf-deps"

	dtFlags1         = 0x6ffffffb
	df1Now           = 0x00000001
	df1NoDefaultLibs = 0x00000800
)

type ELFSym struct {
	elf.ImportedSymbol
}

type ELFLinkDeps struct {
	RPath   []string // DT_RPATH
	RunPath []string // DT_RUNPATH
	Libs    []string
	Flags   ELFLinkFlags
}

type ELFLinkFlags struct {
	Symbolic      bool
	TextRel       bool
	BindNow       bool
	NoDefaultLibs bool
}

type elfPopulator struct{}

func (i *elfPopulator) Name() string {
	return "ELF Populator"
}

func (i *elfPopulator) interp(progs []*elf.Prog) (string, error) {
	for _, p := range progs {
		if p.Type == elf.PT_INTERP {
			d, err := ioutil.ReadAll(p.Open())
			if err != nil {
				return "", fmt.Errorf("reading .interp: %v", err)
			}
			if len(d) > 1 {
				d = d[:len(d)-1]
			}
			return string(d), nil
		}
	}

	return "", errors.New("no .interp section present in binary")
}

func (i *elfPopulator) readFlags(f *elf.File) (ELFLinkFlags, error) {
	var out ELFLinkFlags

	ds := f.SectionByType(elf.SHT_DYNAMIC)
	if ds == nil {
		return out, nil
	}
	d, err := ds.Data()
	if err != nil {
		return out, err
	}

	for len(d) > 0 {
		var t elf.DynTag
		var v uint64
		switch f.Class {
		case elf.ELFCLASS32:
			t = elf.DynTag(f.ByteOrder.Uint32(d[0:4]))
			v = uint64(f.ByteOrder.Uint32(d[4:8]))
			d = d[8:]
		case elf.ELFCLASS64:
			t = elf.DynTag(f.ByteOrder.Uint64(d[0:8]))
			v = f.ByteOrder.Uint64(d[8:16])
			d = d[16:]
		}

		switch t {
		case elf.DT_SYMBOLIC:
			out.Symbolic = v != 0
		case elf.DT_TEXTREL:
			out.TextRel = v != 0
		case elf.DT_BIND_NOW:
			out.BindNow = v != 0

		case elf.DT_FLAGS:
			if v&uint64(elf.DT_SYMBOLIC) != 0 {
				out.Symbolic = true
			}
			if v&uint64(elf.DT_TEXTREL) != 0 {
				out.TextRel = true
			}
			if v&uint64(elf.DT_BIND_NOW) != 0 {
				out.BindNow = true
			}

		case dtFlags1:
			if v&uint64(df1Now) != 0 {
				out.BindNow = true
			}
			if v&uint64(df1NoDefaultLibs) != 0 {
				out.NoDefaultLibs = true
			}
		}
	}
	return out, nil
}

func (i *elfPopulator) Run(t vts.Target, opts *vts.RunnerEnv, info *vts.RuntimeInfo) error {
	r, ok := t.(*vts.Resource)
	if !ok {
		return fmt.Errorf("info.elfPopulator can only operate on resource targets, got %T", t)
	}
	path, err := resourcePath(r)
	f, err := opts.FS.Open(path)
	if err != nil {
		return vts.WrapWithPath(err, path)
	}
	defer f.Close()
	binData, err := elf.NewFile(f)
	if err != nil {
		return vts.WrapWithPath(err, path)
	}

	info.Set(i, ELFHeader, binData.FileHeader)

	s, err := binData.ImportedSymbols()
	if err != nil {
		return vts.WrapWithPath(err, path)
	}
	outSyms := make([]ELFSym, len(s))
	for i, s := range s {
		outSyms[i] = ELFSym{s}
	}
	info.Set(i, ELFDynamicSymbols, outSyms)

	interp, err := i.interp(binData.Progs)
	if err != nil {
		return vts.WrapWithPath(err, path)
	}
	info.Set(i, ELFInterpreter, interp)

	var linkDeps ELFLinkDeps
	libDeps, err := binData.DynString(elf.DT_NEEDED)
	if err != nil {
		return vts.WrapWithPath(err, path)
	}
	linkDeps.Libs = libDeps // Colons are literal - not separators.
	rPath, err := binData.DynString(elf.DT_RPATH)
	if err != nil {
		return vts.WrapWithPath(err, path)
	}
	runPath, err := binData.DynString(elf.DT_RUNPATH)
	if err != nil {
		return vts.WrapWithPath(err, path)
	}
	if linkDeps.Flags, err = i.readFlags(binData); err != nil {
		return vts.WrapWithPath(err, path)
	}
	if runPath != nil {
		linkDeps.RunPath = runPath
	} else {
		linkDeps.RPath = rPath // DT_RPATH ignored if DT_RUNPATH is specified.
	}
	info.Set(i, ELFDeps, linkDeps)

	return nil
}
