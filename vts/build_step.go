package vts

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"sort"

	"go.starlark.net/starlark"
)

type StepKind string

// Valid BuildStep StepKind values.
const (
	StepUnpackGz  = "unpack_gz"
	StepUnpackXz  = "unpack_xz"
	StepShellCmd  = "bash_cmd"
	StepConfigure = "configure"
)

// BuildStep is an anonymous target representing a step in a build.
type BuildStep struct {
	Pos  *DefPosition
	Kind StepKind

	ToPath string
	Path   string
	URL    string
	SHA256 string

	Dir       string
	NamedArgs map[string]string

	Args []string
}

func (t *BuildStep) DefinedAt() *DefPosition {
	return t.Pos
}

func (t *BuildStep) IsClassTarget() bool {
	return false
}

func (t *BuildStep) Validate() error {
	switch t.Kind {
	case StepUnpackGz, StepUnpackXz:
		if t.URL != "" && t.SHA256 == "" {
			return errors.New("sha256 must be specified for all URLs")
		} else if t.Path == "" {
			return errors.New("path or url must be specified")
		}
		if t.ToPath == "" || t.ToPath == "/" {
			return errors.New("to path must specify a destination path")
		}
	case StepShellCmd:
		if len(t.Args) != 1 {
			return errors.New("only one argument can be provided")
		}
	case StepConfigure:
		if t.Dir == "" {
			return errors.New("dir must be specified")
		}
	}
	return nil
}

func (t *BuildStep) String() string {
	return fmt.Sprintf("build_step<%s>", t.Kind)
}

func (t *BuildStep) Freeze() {

}

func (t *BuildStep) Truth() starlark.Bool {
	return true
}

func (t *BuildStep) Type() string {
	return "build_step"
}

func (t *BuildStep) Hash() (uint32, error) {
	h := sha256.Sum256([]byte(fmt.Sprintf("%p", t)))
	return uint32(uint32(h[0]) + uint32(h[1])<<8 + uint32(h[2])<<16 + uint32(h[3])<<24), nil
}

func (t *BuildStep) RollupHash(env *RunnerEnv, eval computeEval) ([]byte, error) {
	hash := sha256.New()
	fmt.Fprintf(hash, "step: %q\n", t.Kind)
	fmt.Fprintf(hash, "%q\n%q\n%q\n%q\n", t.ToPath, t.Path, t.URL, t.SHA256)
	if t.Dir != "" {
		fmt.Fprintf(hash, "dir: %s\n", t.Dir)
	}
	for i, a := range t.Args {
		fmt.Fprintf(hash, "Arg[%d] = %q\n", i, a)
	}
	if len(t.NamedArgs) > 0 {
		ordered := make([]string, 0, len(t.NamedArgs))
		for k := range t.NamedArgs {
			ordered = append(ordered, k)
		}
		sort.Strings(ordered)
		for _, k := range ordered {
			fmt.Fprintf(hash, "NamedArg[%s] = %q\n", k, t.NamedArgs[k])
		}
	}

	return hash.Sum(nil), nil
}
