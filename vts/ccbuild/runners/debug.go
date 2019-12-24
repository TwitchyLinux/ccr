package runners

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"os"

	"github.com/twitchylinux/ccr/vts"
	"go.starlark.net/starlark"
)

// BinutilCheckComponent returns a runner that runs basic sanity checks over
// a component representing a cli binary.
func FailingComponentChecker() *failingCompChecker {
	return &failingCompChecker{}
}

type failingCompChecker struct{}

func (*failingCompChecker) Kind() vts.CheckerKind { return vts.ChkKindEachComponent }

func (*failingCompChecker) String() string { return "debug.failing_component_checker" }

func (*failingCompChecker) Freeze() {}

func (*failingCompChecker) Truth() starlark.Bool { return true }

func (*failingCompChecker) Type() string { return "runner" }

func (t *failingCompChecker) Hash() (uint32, error) {
	h := sha256.Sum256([]byte(fmt.Sprintf("%p", t)))
	return uint32(uint32(h[0]) + uint32(h[1])<<8 + uint32(h[2])<<16 + uint32(h[3])<<24), nil
}

func (r *failingCompChecker) Run(c *vts.Component, opts *vts.RunnerEnv) error {
	return errors.New("debug: returning error")
}

// GenerateDebugManifest returns a generator runner that writes information
// about its inputs to a file.
func GenerateDebugManifest() *debugManifestGenerator {
	return &debugManifestGenerator{}
}

type debugManifestGenerator struct{}

func (*debugManifestGenerator) String() string { return "misc.noop" }

func (*debugManifestGenerator) Freeze() {}

func (*debugManifestGenerator) Truth() starlark.Bool { return true }

func (*debugManifestGenerator) Type() string { return "runner" }

func (t *debugManifestGenerator) Hash() (uint32, error) {
	h := sha256.Sum256([]byte(fmt.Sprintf("%p", t)))
	return uint32(uint32(h[0]) + uint32(h[1])<<8 + uint32(h[2])<<16 + uint32(h[3])<<24), nil
}

func (*debugManifestGenerator) Run(g *vts.Generator, inputs *vts.InputSet, opts *vts.RunnerEnv) error {
	p, err := resourcePath(inputs.Resource)
	if err != nil {
		return err
	}
	f, err := opts.FS.OpenFile(p, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0644)
	if err != nil {
		return vts.WrapWithPath(err, p)
	}
	defer f.Close()

	fmt.Fprintf(f, "Generator: %s\n", g.GlobalPath())
	fmt.Fprintf(f, "Resource: %s\n", inputs.Resource.GlobalPath())
	for _, d := range inputs.Directs {
		fmt.Fprintf(f, "Direct: %T @%s\n", d, d.(vts.GlobalTarget).GlobalPath())
	}
	for class, instances := range inputs.ClassedResources {
		fmt.Fprintf(f, "Class: %s\n", class.GlobalPath())
		for _, inst := range instances {
			fmt.Fprintf(f, "-%s\n", inst.GlobalPath())
		}
	}
	fmt.Fprintln(f)

	return nil
}
