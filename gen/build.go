package gen

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

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

	contractDir string
	fs          billy.Filesystem
	steps       []*vts.BuildStep
}

func (rb *RunningBuild) Dir() string {
	return rb.env.Dir()
}

func (rb *RunningBuild) RootFS() billy.Filesystem {
	return rb.fs
}

func (rb *RunningBuild) SourceFS() billy.Filesystem {
	return osfs.New(rb.contractDir)
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

func (rb *RunningBuild) WriteToCache(c *cache.Cache, b *vts.Build, hash []byte) error {
	fs, err := c.CommitFileset(hash)
	if err != nil {
		return err
	}
	defer fs.Close()

	buildDir, outPathMatcher := rb.env.Dir(), b.OutputMappings()
	err = filepath.Walk(buildDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			// For symlinks: safest thing to do - what if it points outside of the env?
			return nil
		}

		relPath, err := filepath.Rel(buildDir, path)
		if err != nil {
			return err
		}
		if outPath := outPathMatcher.Match(relPath); outPath != "" {
			src, err := os.Open(path)
			if err != nil {
				return err
			}
			if err := fs.AddFile(outPath, info, src); err != nil {
				return err
			}
		}
		return nil
	})
	return err
}

func writeResourceFromBuild(gc GenerationContext, resource *vts.Resource, b *vts.Build, hash []byte) error {
	p, err := determinePath(resource, gc.RunnerEnv)
	if err != nil {
		return err
	}
	m, err := determineMode(resource, gc.RunnerEnv)
	if err != nil && err != errNoAttr {
		return err
	}

	content, closer, defaultMode, err := gc.Cache.FileInFileset(hash, filepath.Base(p))
	if err != nil {
		if err == cache.ErrCacheMiss {
			return err
		}
		if err == os.ErrNotExist {
			return vts.WrapWithPath(vts.WrapWithTarget(vts.WrapWithActionTarget(err, resource), b), p)
		}
		return vts.WrapWithPath(vts.WrapWithTarget(fmt.Errorf("reading from build artifacts: %v", err), resource), p)
	}
	defer closer.Close()

	if m == 0 {
		m = defaultMode
	}
	w, err := gc.RunnerEnv.FS.OpenFile(p, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, m)
	if err != nil {
		return vts.WrapWithPath(err, p)
	}
	defer w.Close()
	if _, err := io.Copy(w, content); err != nil {
		return vts.WrapWithPath(err, p)
	}
	return nil
}

// GenerateBuildSource implements generation of a resource target, based
// on a reference to a build target.
func GenerateBuildSource(gc GenerationContext, resource *vts.Resource, b *vts.Build) error {
	bh, err := b.RollupHash(gc.RunnerEnv, proc.EvalComputedAttribute)
	if err != nil {
		return vts.WrapWithTarget(err, b)
	}

	// Try to read the necessary resource sources from the cache. If that
	// succeeds, theres nothing left to do.
	switch err = writeResourceFromBuild(gc, resource, b, bh); err {
	case nil:
		return nil
	case cache.ErrCacheMiss: // Fallthrough to execute the build.
	default:
		return err
	}

	// If we got this far, the build output is not cached, we need to complete the build manually.
	env, err := proc.NewEnv(false)
	if err != nil {
		return vts.WrapWithTarget(fmt.Errorf("creating build environment: %v", err), b)
	}
	rb := RunningBuild{env: env, steps: b.Steps, fs: osfs.New("/"), contractDir: b.ContractDir}
	if err := rb.Generate(); err != nil {
		rb.Close()
		return vts.WrapWithTarget(fmt.Errorf("build failed: %v", err), b)
	}
	if err := rb.WriteToCache(gc.Cache, b, bh); err != nil {
		rb.Close()
		return vts.WrapWithTarget(fmt.Errorf("gathering output: %v", err), b)
	}
	if err := rb.Close(); err != nil {
		return err
	}

	return writeResourceFromBuild(gc, resource, b, bh)
}
