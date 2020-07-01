package info

import (
	"debug/elf"
	"fmt"

	"github.com/twitchylinux/ccr/vts"
)

// ELF populator data keys.
const (
	ELFHeader = "elf-header"
)

type elfPopulator struct{}

func (i *elfPopulator) Name() string {
	return "ELF Populator"
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
	return nil
}
