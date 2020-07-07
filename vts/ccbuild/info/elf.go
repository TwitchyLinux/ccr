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
)

type ELFSym struct {
	elf.ImportedSymbol
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

	// if path == "/usr/bin" {
	// 	fmt.Println(binData.ImportedSymbols())
	// 	fmt.Println(binData.DynamicSymbols())
	// 	fmt.Println(binData)
	// 	for _, s := range binData.Sections {
	// 		fmt.Printf("Section: %+v\n", s)
	// 	}
	// 	for _, p := range binData.Progs {
	// 		if p.Type == elf.PT_INTERP {
	// 			d, err := ioutil.ReadAll(p.Open())
	// 			if err != nil {
	// 				return fmt.Errorf("reading .interp: %v", err)
	// 			}
	// 			fmt.Printf("Prog: %+v - %s\n", p, string(d))
	// 		}
	// 	}
	// }

	return nil
}
