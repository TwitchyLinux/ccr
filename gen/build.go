package gen

import (
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/twitchylinux/ccr/cache"
	"github.com/twitchylinux/ccr/gen/buildstep"
	"github.com/twitchylinux/ccr/proc"
	"github.com/twitchylinux/ccr/vts"

	"gopkg.in/src-d/go-billy.v4"
	"gopkg.in/src-d/go-billy.v4/osfs"
)

// RunningBuild represents the state for an in-progress build.
type RunningBuild struct {
	env *proc.Env

	sourceDir string
	fs        billy.Filesystem
	steps     []*vts.BuildStep
}

func (rb *RunningBuild) Dir() string {
	return rb.env.Dir()
}

func (rb *RunningBuild) BuildFS() billy.Filesystem {
	return rb.fs
}

func (rb *RunningBuild) SourceFS() billy.Filesystem {
	return osfs.New(rb.sourceDir)
}

func (rb *RunningBuild) Close() error {
	return rb.env.Close()
}

func (rb *RunningBuild) Generate() error {
	for i, step := range rb.steps {
		switch step.Kind {
		case vts.StepUnpackGz:
			if err := buildstep.RunUnpackGz(rb, step); err != nil {
				return fmt.Errorf("step %d (%s) failed: %v", i+1, step.Kind, err)
			}
		default:
			return fmt.Errorf("step %d: unsupported step %s", i+1, step.Kind)
		}
	}
	return nil
}

func applyBuildOutput(buildOutput cache.ReadSeekCloser) error {
	return errors.New("applyBuildOutput() is not yet implemented")
}

// GenerateBuildSource implements generation of a resource target, based
// on a reference to a build target.
func GenerateBuildSource(gc GenerationContext, resource *vts.Resource, b *vts.Build) error {
	bh, err := b.RollupHash(gc.RunnerEnv, proc.EvalComputedAttribute)
	if err != nil {
		return vts.WrapWithTarget(err, b)
	}
	buildOutput, err := gc.Cache.BySHA256(hex.EncodeToString(bh))
	if err != nil && err != cache.ErrCacheMiss {
		return fmt.Errorf("failed cache lookup for %X: %v", bh, err)
	}
	if err == nil {
		return applyBuildOutput(buildOutput)
	}

	// If we got this far, the build output is not cached, we need to complete the build manually.
	env, err := proc.NewEnv(false)
	if err != nil {
		return vts.WrapWithTarget(fmt.Errorf("creating build environment: %v", err), b)
	}
	rb := RunningBuild{env: env, steps: b.Steps, fs: osfs.New("/"), sourceDir: b.Dir}

	if err := rb.Generate(); err != nil {
		rb.Close()
		return vts.WrapWithTarget(fmt.Errorf("build failed: %v", err), b)
	}
	if err := rb.Close(); err != nil {
		return err
	}

	return nil
}
