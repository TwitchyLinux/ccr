package vts

import (
	"crypto/sha256"
	"fmt"
	"strings"

	"github.com/twitchylinux/ccr/vts/match"
	"go.starlark.net/starlark"
)

// Sieve is a target used to filter and combine the output from generators.
type Sieve struct {
	Pos          *DefPosition
	Name         string
	TargetPath   string
	ContractPath string

	Inputs []TargetRef

	AddPrefix    string
	Renames      *match.FilenameRules
	ExcludeGlobs []string
	IncludeGlobs []string
}

func (t *Sieve) DefinedAt() *DefPosition {
	return t.Pos
}

func (t *Sieve) IsClassTarget() bool {
	return false
}

func (t *Sieve) GlobalPath() string {
	return t.TargetPath
}

func (t *Sieve) TargetName() string {
	return t.Name
}

func (t *Sieve) TargetType() TargetType {
	return TargetSieve
}

// IsPrefixSieve returns true if the sieve matches a single subset of
// files, and trims the same prefix from their paths.
func (t *Sieve) IsDirPrefixSieve() bool {
	if len(t.IncludeGlobs) != 1 || t.Renames == nil || len(t.Renames.Rules) != 1 || len(t.ExcludeGlobs) != 0 {
		return false
	}
	if !strings.HasSuffix(t.IncludeGlobs[0], "/**") {
		return false
	}
	if spm, isSpm := t.Renames.Rules[0].Out.(*match.StripPrefixOutputMapper); isSpm {
		return spm.Prefix == strings.TrimSuffix(t.IncludeGlobs[0], "**")
	}
	return false
}

// DirPrefix returns the prefix filtered and trimmed by the sieve. The result
// is undefined if IsDirPrefixSieve() != true.
func (t *Sieve) DirPrefix() string {
	if len(t.IncludeGlobs) != 1 {
		return ""
	}
	return strings.TrimSuffix(t.IncludeGlobs[0], "/**")
}

func (t *Sieve) Validate() error {
	for i, inp := range t.Inputs {
		_, puesdo := inp.Target.(*Puesdo)
		_, build := inp.Target.(*Build)
		_, sieve := inp.Target.(*Sieve)
		if !puesdo && !build && !sieve {
			return fmt.Errorf("inputs[%d] is type %T, but must be build, sieve, file, or deb", i, inp.Target)
		}
		if len(inp.Constraints) > 0 {
			return fmt.Errorf("inputs[%d]: cannot specify constraints here", i)
		}
	}
	return nil
}

func (t *Sieve) NeedInputs() []TargetRef {
	return t.Inputs
}

func (t *Sieve) String() string {
	return fmt.Sprintf("sieve<%s>", t.Name)
}

func (t *Sieve) Freeze() {

}

func (t *Sieve) Truth() starlark.Bool {
	return true
}

func (t *Sieve) Type() string {
	return "sieve"
}

func (t *Sieve) Hash() (uint32, error) {
	h := sha256.Sum256([]byte(fmt.Sprintf("%p", t)))
	return uint32(uint32(h[0]) + uint32(h[1])<<8 + uint32(h[2])<<16 + uint32(h[3])<<24), nil
}

func (t *Sieve) RollupHash(env *RunnerEnv, eval computeEval) ([]byte, error) {
	hash := sha256.New()
	fmt.Fprintf(hash, "Sieve: %q\n%s\n", t.Name, t.AddPrefix)

	for _, dep := range t.Inputs {
		rt, isHashable := dep.Target.(ReproducibleTarget)
		if !isHashable {
			return nil, WrapWithTarget(fmt.Errorf("cannot compute rollup hash on non-reproducible target of type %T", dep.Target), dep.Target)
		}
		h, err := rt.RollupHash(env, eval)
		if err != nil {
			return nil, err
		}
		hash.Write(h)
	}
	for _, ex := range t.ExcludeGlobs {
		fmt.Fprintf(hash, "Ex pattern: %s\n", ex)
	}
	for _, ex := range t.IncludeGlobs {
		fmt.Fprintf(hash, "Inc pattern: %s\n", ex)
	}
	if t.Renames != nil {
		fmt.Fprintln(hash, t.Renames.RollupHash())
	}

	return hash.Sum(nil), nil
}
