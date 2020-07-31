package gen

import (
	"archive/tar"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/twitchylinux/ccr/cache"
	"github.com/twitchylinux/ccr/gen/buildstep"
	"github.com/twitchylinux/ccr/proc"
	"github.com/twitchylinux/ccr/vts"
	"github.com/twitchylinux/ccr/vts/common"
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

func (rb *RunningBuild) OverlayMountPath() string {
	return rb.env.OverlayMountPath()
}

func (rb *RunningBuild) OverlayUpperPath() string {
	return rb.env.OverlayUpperPath()
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

func (rb *RunningBuild) ExecBlocking(args []string, stdout, stderr io.Writer) (int, error) {
	id, err := rb.env.RunStreaming("/tmp", stdout, stderr, args...)
	if err != nil {
		return 0, err
	}
	if err := rb.env.WaitStreaming(id); err != nil {
		return 0, err
	}
	return rb.env.StreamingExitCode(id), nil
}

func (rb *RunningBuild) EnsurePatched(path string) error {
	pathSegments := strings.Split(strings.TrimPrefix(path, "/"), string(filepath.Separator))
	return rb.env.EnsurePatched(pathSegments[0])
}

func (rb *RunningBuild) Patch(gc GenerationContext, patches map[string]vts.TargetRef) error {
	for path, patch := range patches {
		fsr, err := filesetForSource(gc, patch.Target)
		if err != nil {
			return err
		}
		if err := rb.patchToPath(gc, path, patch.Target, fsr); err != nil {
			fsr.Close()
			return err
		}
		if err := fsr.Close(); err != nil {
			return fmt.Errorf("closing fileset: %v", err)
		}

		if err := rb.EnsurePatched(path); err != nil {
			return err
		}
	}
	return nil
}

func (rb *RunningBuild) patchToPath(gc GenerationContext, path string, pt vts.Target, fsr fileset) error {
	switch t := pt.(type) {
	case *vts.Build:
		return writeMultiFilesFromBuild(gc.Cache, rb.fs, filepath.Join(rb.OverlayUpperPath(), path), fsr)

	case *vts.Puesdo:
		switch t.Kind {
		case vts.FileRef:
			oPath := filepath.Join(rb.OverlayUpperPath(), path)
			if err := os.MkdirAll(filepath.Dir(oPath), 0755); err != nil && !os.IsExist(err) {
				return vts.WrapWithPath(err, path)
			}
			return populateFileToPath(rb.fs, fsr, oPath, 0, nil)

		default:
			return vts.WrapWithActionTarget(fmt.Errorf("cannot patch from puesdo-target of kind %v", t.Kind), t)
		}
	}

	return vts.WrapWithPath(fmt.Errorf("cannot patch using source target of type %T", pt), path)
}

func (rb *RunningBuild) Generate(c *cache.Cache) error {
	for i, step := range rb.steps {
		switch step.Kind {
		case vts.StepUnpackGz, vts.StepUnpackXz:
			if err := buildstep.RunUnpack(c, rb, step); err != nil {
				return fmt.Errorf("step %d (%s) failed: %v", i+1, step.Kind, err)
			}
			if err := rb.EnsurePatched(step.ToPath); err != nil {
				return fmt.Errorf("step %d (%s) failed wiring into filesystem: %v", i+1, step.Kind, err)
			}
		case vts.StepShellCmd:
			if err := buildstep.RunShellCmd(rb, step); err != nil {
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

	buildDir, outPathMatcher := rb.env.OverlayUpperPath(), b.OutputMappings()
	err = filepath.Walk(buildDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			// For symlinks: safest thing to do - what if it points outside of the env?
			return nil
		}
		if strings.HasPrefix(filepath.Base(path), ".wh.") {
			return nil // ignore whiteout markers from fuse-overlayfs.
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

func writeResourceFromBuild(gc GenerationContext, resource *vts.Resource, fsr fileset) error {
	outPath, mode, err := resourcePathMode(resource, gc.RunnerEnv)
	if err != nil {
		return err
	}

	parent := resource.Parent.Target.(*vts.ResourceClass)
	switch parent {
	case common.FileResourceClass:
		err = populateFileToPath(gc.RunnerEnv.FS, fsr, outPath, mode, func(path string, _ *tar.Header) (bool, error) {
			return path != filepath.Base(outPath), nil
		})
	case common.CHeadersResourceClass:
		err = writeMultiFilesFromBuild(gc.Cache, gc.RunnerEnv.FS, outPath, fsr)
	default:
		return fmt.Errorf("cannot populate from build for resources of class %q", parent.GlobalPath())
	}

	if err != nil {
		if err == os.ErrNotExist {
			err = errors.New("file missing from build output")
		}
		return vts.WrapWithPath(vts.WrapWithTarget(err, resource), outPath)
	}
	return nil
}

func writeMultiFilesFromBuild(c *cache.Cache, fs billy.Filesystem, p string, fr fileset) error {
	for {
		path, h, err := fr.Next()
		if err != nil {
			if err == io.EOF {
				break
			}
			return fmt.Errorf("iterating build fileset: %v", err)
		}

		switch h.Typeflag {
		case tar.TypeDir:
			if err := fs.MkdirAll(filepath.Join(p, path), 0755); err != nil {
				return vts.WrapWithPath(fmt.Errorf("mkdir from fileset: %v", err), path)
			}

		case tar.TypeReg:
			outFile, err := fs.OpenFile(filepath.Join(p, path), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, h.FileInfo().Mode())
			if err != nil {
				return vts.WrapWithPath(fmt.Errorf("open from fileset: %v", err), path)
			}
			if _, err := io.Copy(outFile, fr); err != nil {
				outFile.Close()
				return vts.WrapWithPath(fmt.Errorf("copying from fileset: %v", err), path)
			}
			outFile.Close()
		}
	}

	return nil
}

// generateBuild executes a build if the result is not already cached.
func generateBuild(gc GenerationContext, b *vts.Build) error {
	bh, err := b.RollupHash(gc.RunnerEnv, proc.EvalComputedAttribute)
	if err != nil {
		return vts.WrapWithTarget(err, b)
	}
	// See if its already cached.
	if _, err := os.Stat(gc.Cache.SHA256Path(hex.EncodeToString(bh))); err == nil {
		return nil
	}

	// If we got this far, the build output is not cached, we need to complete the build manually.
	env, err := proc.NewEnv(false)
	if err != nil {
		return vts.WrapWithTarget(fmt.Errorf("creating build environment: %v", err), b)
	}
	rb := RunningBuild{env: env, steps: b.Steps, fs: osfs.New("/"), contractDir: b.ContractDir}
	if err := rb.Patch(gc, b.PatchIns); err != nil {
		rb.Close()
		return vts.WrapWithTarget(fmt.Errorf("failed to apply patch-ins: %v", err), b)
	}
	if err := rb.Generate(gc.Cache); err != nil {
		rb.Close()
		return vts.WrapWithTarget(fmt.Errorf("build failed: %v", err), b)
	}
	if err := rb.WriteToCache(gc.Cache, b, bh); err != nil {
		rb.Close()
		return vts.WrapWithTarget(fmt.Errorf("gathering output: %v", err), b)
	}
	return rb.Close()
}
