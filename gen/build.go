package gen

import (
	"archive/tar"
	"compress/gzip"
	"encoding/hex"
	"errors"
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

func (rb *RunningBuild) BuildFS() billy.Filesystem {
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

func (rb *RunningBuild) WriteOutput(c *cache.Cache, b *vts.Build, hash []byte) error {
	// TODO: Move all this into methods on cache.
	f, err := os.OpenFile(c.SHA256Path(hex.EncodeToString(hash)), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer f.Close()
	gzipW := gzip.NewWriter(f)
	defer gzipW.Close()
	tarW := tar.NewWriter(gzipW)
	defer tarW.Close()

	buildDir, outPathMatcher := rb.env.Dir(), b.OutputMappings()
	err = filepath.Walk(buildDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return nil // safest thing to do - what if it points outside of the env?
		}
		if info.IsDir() {
			return nil
		}

		var relPath string
		if relPath, err = filepath.Rel(buildDir, path); err != nil {
			return err
		}
		if outPath := outPathMatcher.Match(relPath); outPath != "" {
			if err := tarW.WriteHeader(&tar.Header{
				Name:    outPath,
				Size:    info.Size(),
				Mode:    int64(info.Mode()),
				ModTime: info.ModTime(),
			}); err != nil {
				return fmt.Errorf("%q: writing header: %v", relPath, err)
			}
			src, err := os.Open(path)
			if err != nil {
				return err
			}
			if _, err := io.Copy(tarW, src); err != nil {
				return fmt.Errorf("%q: copy content: %v", relPath, err)
			}
			if err := src.Close(); err != nil {
				return err
			}
		}
		return nil
	})
	return err
}

func applyBuildOutput(gc GenerationContext, resource *vts.Resource, buildOutput cache.ReadSeekCloser) error {
	p, err := determinePath(resource, gc.RunnerEnv)
	if err != nil {
		return err
	}
	m, err := determineMode(resource, gc.RunnerEnv)
	if err != nil && err != errNoAttr {
		return err
	}

	tape, err := gzip.NewReader(buildOutput)
	if err != nil {
		return fmt.Errorf("reading gzip: %v", err)
	}
	tr := tar.NewReader(tape)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("reading tar: %v", err)
		}

		switch header.Typeflag {
		case tar.TypeDir:
		case tar.TypeReg:
			if header.Name == filepath.Base(p) {
				if m == 0 {
					m = os.FileMode(header.Mode)
				}
				w, err := gc.RunnerEnv.FS.OpenFile(p, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, m)
				if err != nil {
					return vts.WrapWithPath(err, p)
				}
				defer w.Close()
				if _, err := io.Copy(w, tr); err != nil {
					return vts.WrapWithPath(err, p)
				}
				return nil
			}
		default:
			return fmt.Errorf("unsupported tar resource: %x", header.Typeflag)
		}
	}

	return vts.WrapWithPath(vts.WrapWithTarget(errors.New("could not find file in build artifacts"), resource), p)
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
		// Cache hit, this build has been completed before! Lets write out
		// the artifacts this resource needs and call it a day.
		defer buildOutput.Close()
		return applyBuildOutput(gc, resource, buildOutput)
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
	if err := rb.WriteOutput(gc.Cache, b, bh); err != nil {
		rb.Close()
		return vts.WrapWithTarget(fmt.Errorf("gathering output: %v", err), b)
	}
	if err := rb.Close(); err != nil {
		return err
	}

	// Build is completed and artifacts written to cache.
	if buildOutput, err = gc.Cache.BySHA256(hex.EncodeToString(bh)); err != nil {
		return fmt.Errorf("reading build artifacts %X: %v", bh, err)
	}
	defer buildOutput.Close()

	return applyBuildOutput(gc, resource, buildOutput)
}
