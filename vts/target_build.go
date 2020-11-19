package vts

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"sort"

	"github.com/twitchylinux/ccr/vts/match"
	"go.starlark.net/starlark"
)

const buildOutputHashCacheBuster = 3

// Build is a target representing a build.
type Build struct {
	Path         string
	Name         string
	Pos          *DefPosition
	ContractDir  string
	ContractPath string

	HostDeps       []TargetRef
	Steps          []*BuildStep
	Output         *match.FilenameRules
	PatchIns       map[string]TargetRef
	Injections     []TargetRef
	Env            map[string]starlark.Value
	UsingRoot      *TargetRef
	ProducesRootFS bool

	cachedRollupHash []byte
}

func (t *Build) DefinedAt() *DefPosition {
	return t.Pos
}

func (t *Build) IsClassTarget() bool {
	return false
}

func (t *Build) TargetType() TargetType {
	return TargetBuild
}

func (t *Build) GlobalPath() string {
	return t.Path
}

func (t *Build) TargetName() string {
	return t.Name
}

func (t *Build) Validate() error {
	if t.UsingRoot != nil {
		if t.UsingRoot.Target != nil {
			b, ok := t.UsingRoot.Target.(*Build)
			if !ok {
				return fmt.Errorf("using_chroot is type %T, but must be type %T", t.UsingRoot.Target, t)
			}
			if !b.ProducesRootFS {
				return errors.New("target referenced as chroot does not set root_fs")
			}
		}
	}
	return nil
}

func (t *Build) HostDependencies() []TargetRef {
	return t.HostDeps
}

func (t *Build) NeedInputs() []TargetRef {
	out := make([]TargetRef, 0, len(t.PatchIns)+len(t.Injections))
	for _, t := range t.PatchIns {
		out = append(out, t)
	}
	for _, t := range t.Injections {
		out = append(out, t)
	}
	return out
}

func (t *Build) UsingRootFS() *TargetRef {
	return t.UsingRoot
}

func (t *Build) String() string {
	if t.Name == "" {
		return "build<$anonymous$>"
	}
	return fmt.Sprintf("build<%s>", t.Name)
}

func (t *Build) Freeze() {

}

func (t *Build) Truth() starlark.Bool {
	return true
}

func (t *Build) Type() string {
	return "build"
}

func (t *Build) Hash() (uint32, error) {
	h := sha256.Sum256([]byte(fmt.Sprintf("%p", t)))
	return uint32(uint32(h[0]) + uint32(h[1])<<8 + uint32(h[2])<<16 + uint32(h[3])<<24), nil
}

func (t *Build) RollupHash(env *RunnerEnv, eval computeEval) ([]byte, error) {
	if t.cachedRollupHash != nil {
		return t.cachedRollupHash, nil
	}

	hash := sha256.New()
	fmt.Fprintf(hash, "%d-build: %q\n%q\n", buildOutputHashCacheBuster, t.Path, t.Name)

	for _, dep := range t.HostDeps {
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
	for _, step := range t.Steps {
		h, err := step.RollupHash(env, eval)
		if err != nil {
			return nil, err
		}
		hash.Write(h)
	}

	if t.PatchIns != nil {
		fmt.Fprintln(hash, "Input patches:")
		out := make([]string, 0, len(t.PatchIns))
		for path, t := range t.PatchIns {
			rt, isHashable := t.Target.(ReproducibleTarget)
			if !isHashable {
				return nil, WrapWithTarget(fmt.Errorf("cannot compute rollup hash on non-reproducible target of type %T", t.Target), t.Target)
			}
			h, err := rt.RollupHash(env, eval)
			if err != nil {
				return nil, err
			}
			out = append(out, fmt.Sprintf("%s: %X", path, h))
		}
		sort.Strings(out)
		for _, kv := range out {
			hash.Write([]byte(kv))
		}
	}
	if t.Output != nil {
		fmt.Fprintln(hash, t.Output.RollupHash())
	}

	for _, inj := range t.Injections {
		rt, isHashable := inj.Target.(ReproducibleTarget)
		if !isHashable {
			return nil, WrapWithTarget(fmt.Errorf("cannot compute rollup hash on non-reproducible target of type %T", inj.Target), inj.Target)
		}
		h, err := rt.RollupHash(env, eval)
		if err != nil {
			return nil, err
		}
		hash.Write(h)
	}
	if len(t.Env) > 0 {
		ordered := make([]string, 0, len(t.Env))
		for k := range t.Env {
			ordered = append(ordered, k)
		}
		sort.Strings(ordered)
		for _, k := range ordered {
			fmt.Fprintf(hash, "Env[%s] = %q\n", k, t.Env[k].String())
		}
	}

	if t.ProducesRootFS {
		fmt.Fprintln(hash, "RootFS = true")
	}
	if t.UsingRoot != nil {
		rt, isHashable := t.UsingRoot.Target.(ReproducibleTarget)
		if !isHashable {
			return nil, WrapWithTarget(fmt.Errorf("cannot compute rollup hash on non-reproducible target of type %T", t.UsingRoot), t.UsingRoot.Target)
		}
		h, err := rt.RollupHash(env, eval)
		if err != nil {
			return nil, err
		}
		hash.Write(h)
	}

	t.cachedRollupHash = hash.Sum(nil)
	return t.cachedRollupHash, nil
}

func (t *Build) OutputMappings() *match.FilenameRules {
	return t.Output
}
