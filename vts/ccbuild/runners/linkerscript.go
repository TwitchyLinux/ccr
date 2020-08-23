package runners

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/twitchylinux/ccr/proc"
	"github.com/twitchylinux/ccr/vts"
	"github.com/twitchylinux/ccr/vts/ccbuild/info"
	"go.starlark.net/starlark"
)

// GenerateUnionLinkerscript returns a generator runner that generates
// a union linkerscript.
func GenerateUnionLinkerscript() *unionLinkerscriptGenerator {
	return &unionLinkerscriptGenerator{}
}

type unionLinkerscriptGenerator struct{}

func (*unionLinkerscriptGenerator) String() string { return "linkerscript.union.generator" }

func (*unionLinkerscriptGenerator) Freeze() {}

func (*unionLinkerscriptGenerator) Truth() starlark.Bool { return true }

func (*unionLinkerscriptGenerator) Type() string { return "runner" }

func (t *unionLinkerscriptGenerator) Hash() (uint32, error) {
	h := sha256.Sum256([]byte(fmt.Sprintf("%p", t)))
	return uint32(uint32(h[0]) + uint32(h[1])<<8 + uint32(h[2])<<16 + uint32(h[3])<<24), nil
}

func (*unionLinkerscriptGenerator) Run(g *vts.Generator, inputs *vts.InputSet, opts *vts.RunnerEnv) error {
	p, err := resourcePath(inputs.Resource, opts)
	if err != nil {
		return err
	}
	libs, err := resourceLdInputs(inputs.Resource, opts)
	if err != nil {
		if err == errNoAttr {
			return errors.New("cannot generate symlink when no target was specified")
		}
		return err
	}

	var libStrings []string
	for _, lib := range libs {
		if opts.Universe != nil {
			// Load the library resource pointed to by the input, and read ELF
			// information. This should effectively verify its a valid ELF library.
			libR, err := opts.Universe.FindByPath(lib, opts)
			if err != nil {
				return vts.WrapWithPath(fmt.Errorf("finding input library %q: %v", lib, err), lib)
			}
			libRes, ok := libR.(*vts.Resource)
			if !ok {
				return vts.WrapWithPath(fmt.Errorf("expected resource target at path, got %T", libR), lib)
			}

			ri := libRes.RuntimeInfo()
			if ri.HasRun(info.ELFPopulator) {
				continue
			}
			if err := info.ELFPopulator.Run(libRes, opts, ri); err != nil {
				err = vts.WrapWithTarget(err, libRes)
				err = vts.WrapWithActionTarget(err, g)
				return err
			}
			if _, err := ri.Get(info.ELFPopulator, info.ELFHeader); err != nil {
				return vts.WrapWithPath(err, lib)
			}
		}

		b := filepath.Base(lib)
		if strings.HasPrefix(b, "lib") && filepath.Ext(b) == "so" {
			libStrings = append(libStrings, "-l"+strings.Split(b, ".")[0][3:])
		} else if filepath.Dir(lib) == filepath.Dir(p) {
			libStrings = append(libStrings, b)
		} else {
			return fmt.Errorf("cannot use input %q", lib)
		}
	}

	var out strings.Builder
	out.WriteString("INPUT(")
	out.WriteString(strings.Join(libStrings, " "))
	out.WriteString(")\n")

	f, err := opts.FS.OpenFile(p, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0755)
	if err != nil {
		return vts.WrapWithPath(err, p)
	}
	if _, err := io.Copy(f, bytes.NewBufferString(out.String())); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}

func resourceLdInputs(r *vts.Resource, env *vts.RunnerEnv) ([]string, error) {
	var out []string
	for _, attr := range r.Details {
		if attr.Target == nil {
			return nil, fmt.Errorf("unresolved target reference: %q", attr.Path)
		}
		a := attr.Target.(*vts.Attr)
		if a.Parent.Target == nil {
			return nil, fmt.Errorf("unresolved target reference: %q", a.Parent.Path)
		}
		if class := a.Parent.Target.(*vts.AttrClass); class.GlobalPath() == "common://attrs:ldscript_input_library" {
			v, err := a.Value(r, env, proc.EvalComputedAttribute)
			if err != nil {
				return nil, err
			}
			if s, ok := v.(starlark.String); ok {
				out = append(out, string(s))
			} else {
				return nil, fmt.Errorf("expected string value, got %T", v)
			}
		}
	}

	if len(out) == 0 {
		return nil, errNoAttr
	}
	return out, nil
}
