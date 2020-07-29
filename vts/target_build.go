package vts

import (
	"crypto/sha256"
	"fmt"
	"sort"
	"strings"

	"github.com/gobwas/glob"
	"go.starlark.net/starlark"
)

const buildOutputHashCacheBuster = 1

// Build is a target representing a build.
type Build struct {
	Path         string
	Name         string
	Pos          *DefPosition
	ContractDir  string
	ContractPath string

	HostDeps []TargetRef
	Steps    []*BuildStep
	Output   *starlark.Dict
	PatchIns map[string]TargetRef
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
	return nil
}

func (t *Build) HostDependencies() []TargetRef {
	return t.HostDeps
}

func (t *Build) NeedInputs() []TargetRef {
	out := make([]TargetRef, 0, len(t.PatchIns))
	for _, t := range t.PatchIns {
		out = append(out, t)
	}
	return out
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
		fmt.Fprintln(hash, "Output mappings:")
		out := make([]string, 0, t.Output.Len())
		for _, e := range t.Output.Items() {
			out = append(out, fmt.Sprintf("%s: %s", e[0].String(), e[1].String()))
		}
		sort.Strings(out)
		for _, kv := range out {
			hash.Write([]byte(kv))
		}
	}

	return hash.Sum(nil), nil
}

type artifactMatch struct {
	p   glob.Glob
	out BuildOutputMapper
}

type BuildArtifactMatcher struct {
	rules []artifactMatch
}

func (m *BuildArtifactMatcher) Match(artifactPath string) string {
	for _, r := range m.rules {
		if r.p.Match(artifactPath) {
			return r.out.Map(artifactPath)
		}
	}
	return ""
}

func (t *Build) OutputMappings() BuildArtifactMatcher {
	if t.Output == nil {
		return BuildArtifactMatcher{}
	}
	out := BuildArtifactMatcher{rules: make([]artifactMatch, t.Output.Len())}
	keys := make([]string, 0, t.Output.Len())
	for _, e := range t.Output.Keys() {
		keys = append(keys, string(e.(starlark.String)))
	}
	sort.Strings(keys)

	for i, k := range keys {
		v, _, _ := t.Output.Get(starlark.String(k))
		var mapper BuildOutputMapper
		if s, isStr := v.(starlark.String); isStr {
			mapper = LiteralOutputMapper(s)
		} else {
			mapper = v.(BuildOutputMapper)
		}

		out.rules[i] = artifactMatch{p: glob.MustCompile(strings.TrimPrefix(k, "/")), out: mapper}
	}
	return out
}
