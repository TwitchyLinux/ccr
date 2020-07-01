package runners

import (
	"crypto/sha256"
	"debug/elf"
	"fmt"
	"os"

	"github.com/twitchylinux/ccr/vts"
	"github.com/twitchylinux/ccr/vts/ccbuild/info"
	"go.starlark.net/starlark"
)

// BinaryCheckValid returns a runner that can check resources
// reference a well-formed executable binary file.
func BinaryCheckValid() *binaryValidRunner {
	return &binaryValidRunner{}
}

type binaryValidRunner struct{}

func (*binaryValidRunner) Kind() vts.CheckerKind { return vts.ChkKindEachResource }

func (*binaryValidRunner) String() string { return "binary.check_valid" }

func (*binaryValidRunner) Freeze() {}

func (*binaryValidRunner) Truth() starlark.Bool { return true }

func (*binaryValidRunner) Type() string { return "runner" }

func (t *binaryValidRunner) Hash() (uint32, error) {
	h := sha256.Sum256([]byte(fmt.Sprintf("%p", t)))
	return uint32(uint32(h[0]) + uint32(h[1])<<8 + uint32(h[2])<<16 + uint32(h[3])<<24), nil
}

func (*binaryValidRunner) Run(resource *vts.Resource, chkr *vts.Checker, opts *vts.RunnerEnv) error {
	d, err := resource.RuntimeInfo().Get(info.StatPopulator, info.FileStat)
	if err != nil {
		return err
	}
	fileInfo := d.(info.FileInfo)

	if d, err = resource.RuntimeInfo().Get(info.ELFPopulator, info.ELFHeader); err != nil {
		return err
	}
	elfHeader := d.(elf.FileHeader)

	// Sanity checks.
	if elfHeader.Data != elf.ELFDATA2LSB { // TODO: Parameterize by config.
		return vts.WrapWithPath(fmt.Errorf("elf data is not %v, got %v", elf.ELFDATA2LSB, elfHeader.Data), fileInfo.Path)
	}
	if elfHeader.Version != elf.EV_CURRENT {
		return vts.WrapWithPath(fmt.Errorf("elf version is not %v, got %v", elf.EV_CURRENT, elfHeader.Version), fileInfo.Path)
	}
	if elfHeader.Class != elf.ELFCLASS32 && elfHeader.Class != elf.ELFCLASS64 {
		return vts.WrapWithPath(fmt.Errorf("elf class is not 32/64, got %v", elfHeader.Class), fileInfo.Path)
	}
	if elfHeader.OSABI != elf.ELFOSABI_NONE && elfHeader.OSABI != elf.ELFOSABI_LINUX {
		return vts.WrapWithPath(fmt.Errorf("elf ABI is non-linux %v", elfHeader.OSABI), fileInfo.Path)
	}
	if elfHeader.Type != elf.ET_EXEC {
		return vts.WrapWithPath(fmt.Errorf("elf type is non-exec %v", elfHeader.Type), fileInfo.Path)
	}
	if elfHeader.Machine != elf.EM_X86_64 && elfHeader.Machine != elf.EM_386 { // TODO: Parameterize by config.
		return vts.WrapWithPath(fmt.Errorf("elf arch is %v", elfHeader.Machine), fileInfo.Path)
	}

	// for _, section := range binData.Sections {
	// 	fmt.Printf("section: %+v\n", section)
	// }
	// syms, _ := binData.ImportedSymbols()
	// things, _ := binData.DynamicSymbols()
	// libs, _ := binData.ImportedLibraries()
	// fmt.Printf("syms: %+v\n", syms)
	// fmt.Printf("dyns: %+v\n", things)
	// fmt.Printf("libs: %+v\n", libs)

	if fileInfo.Mode()&os.ModePerm&0111 == 0 {
		return vts.WrapWithPath(fmt.Errorf("binary is not executable: %#o", fileInfo.Mode()), fileInfo.Path)
	}

	return nil
}

func (*binaryValidRunner) PopulatorsNeeded() []vts.InfoPopulator {
	return []vts.InfoPopulator{info.StatPopulator, info.ELFPopulator}
}
