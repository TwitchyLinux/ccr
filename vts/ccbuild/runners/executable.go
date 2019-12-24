package runners

import (
	"crypto/sha256"
	"debug/elf"
	"errors"
	"fmt"
	"os"

	"github.com/twitchylinux/ccr/vts"
	"go.starlark.net/starlark"
)

// BinutilCheckComponent returns a runner that runs basic sanity checks over
// a component representing a cli binary.
func BinutilCheckComponent() *binutilCheckRunner {
	return &binutilCheckRunner{}
}

type binutilCheckRunner struct{}

func (*binutilCheckRunner) Kind() vts.CheckerKind { return vts.ChkKindEachComponent }

func (*binutilCheckRunner) String() string { return "binutil.sanity_check" }

func (*binutilCheckRunner) Freeze() {}

func (*binutilCheckRunner) Truth() starlark.Bool { return true }

func (*binutilCheckRunner) Type() string { return "runner" }

func (t *binutilCheckRunner) Hash() (uint32, error) {
	h := sha256.Sum256([]byte(fmt.Sprintf("%p", t)))
	return uint32(uint32(h[0]) + uint32(h[1])<<8 + uint32(h[2])<<16 + uint32(h[3])<<24), nil
}

func (r *binutilCheckRunner) Run(c *vts.Component, opts *vts.RunnerEnv) error {
	return errors.New("not implemented: " + r.String())
}

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

func (*binaryValidRunner) Run(resource *vts.Resource, opts *vts.RunnerEnv) error {
	path, err := resourcePath(resource)
	if err != nil {
		return err
	}
	f, err := opts.FS.Open(path)
	if err != nil {
		return vts.WrapWithPath(err, path)
	}
	defer f.Close()
	binData, err := elf.NewFile(f)
	if err != nil {
		return vts.WrapWithPath(err, path)
	}

	// Sanity checks.
	if binData.Data != elf.ELFDATA2LSB { // TODO: Parameterize by config.
		return vts.WrapWithPath(fmt.Errorf("elf data is not %v, got %v", elf.ELFDATA2LSB, binData.Data), path)
	}
	if binData.Version != elf.EV_CURRENT {
		return vts.WrapWithPath(fmt.Errorf("elf version is not %v, got %v", elf.EV_CURRENT, binData.Version), path)
	}
	if binData.Class != elf.ELFCLASS32 && binData.Class != elf.ELFCLASS64 {
		return vts.WrapWithPath(fmt.Errorf("elf class is not 32/64, got %v", binData.Class), path)
	}
	if binData.OSABI != elf.ELFOSABI_NONE && binData.OSABI != elf.ELFOSABI_LINUX {
		return vts.WrapWithPath(fmt.Errorf("elf ABI is non-linux %v", binData.OSABI), path)
	}
	if binData.Type != elf.ET_EXEC {
		return vts.WrapWithPath(fmt.Errorf("elf type is non-exec %v", binData.Type), path)
	}
	if binData.Machine != elf.EM_X86_64 && binData.Machine != elf.EM_386 { // TODO: Parameterize by config.
		return vts.WrapWithPath(fmt.Errorf("elf arch is %v", binData.Machine), path)
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

	s, err := opts.FS.Stat(path)
	if err != nil {
		return vts.WrapWithPath(err, path)
	}
	if s.Mode()&os.ModePerm&0111 == 0 {
		return vts.WrapWithPath(fmt.Errorf("binary is not executable: %#o", s.Mode()), path)
	}

	return nil
}
