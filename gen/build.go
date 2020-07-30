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
	// TODO: Move most of this logic into the proc package.
	for path, patch := range patches {
		switch t := patch.Target.(type) {
		case *vts.Build:
			h, err := t.RollupHash(gc.RunnerEnv, proc.EvalComputedAttribute)
			if err != nil {
				return vts.WrapWithTarget(err, t)
			}
			if err := writeMultiFilesFromBuild(gc.Cache, rb.fs, filepath.Join(rb.OverlayUpperPath(), path), t, h); err != nil {
				return err
			}

		case *vts.Puesdo:
			switch t.Kind {
			case vts.FileRef:
				srcFilePath := filepath.Join(filepath.Dir(t.ContractPath), t.Path)
				if t.Host {
					srcFilePath = t.Path
				}
				s, err := os.Stat(srcFilePath)
				if err != nil {
					return vts.WrapWithPath(err, path)
				}

				oPath := filepath.Join(rb.OverlayUpperPath(), path)
				if err := os.MkdirAll(filepath.Dir(oPath), 0755); err != nil && !os.IsExist(err) {
					return vts.WrapWithPath(err, path)
				}
				if err := populateFileToPath(rb.fs, srcFilePath, oPath, s.Mode()); err != nil {
					return vts.WrapWithPath(err, path)
				}

			default:
				return vts.WrapWithActionTarget(fmt.Errorf("cannot patch from puesdo-target of kind %v", t.Kind), t)
			}
		}

		if err := rb.EnsurePatched(path); err != nil {
			return err
		}
	}
	return nil
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

func writeResourceFromBuild(gc GenerationContext, resource *vts.Resource, b *vts.Build, hash []byte) error {
	parent := resource.Parent.Target.(*vts.ResourceClass)
	switch parent {
	case common.FileResourceClass:
		return writeFileResourceFromBuild(gc, resource, b, hash)
	case common.CHeadersResourceClass:
		p, err := determinePath(resource, gc.RunnerEnv)
		if err != nil {
			return err
		}
		if err := writeMultiFilesFromBuild(gc.Cache, gc.RunnerEnv.FS, p, b, hash); err != nil {
			return vts.WrapWithTarget(err, resource)
		}
		return nil
	}
	return fmt.Errorf("cannot populate from build for resources of class %q", parent.GlobalPath())
}

func writeMultiFilesFromBuild(c *cache.Cache, fs billy.Filesystem, p string, b *vts.Build, hash []byte) error {
	fr, err := c.FilesetReader(hash)
	if err != nil {
		if err == cache.ErrCacheMiss {
			return err
		}
		return vts.WrapWithPath(fmt.Errorf("reading from build artifacts: %v", err), p)
	}
	defer fr.Close()

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

func writeFileResourceFromBuild(gc GenerationContext, resource *vts.Resource, b *vts.Build, hash []byte) error {
	p, err := determinePath(resource, gc.RunnerEnv)
	if err != nil {
		return err
	}
	m, err := determineMode(resource, gc.RunnerEnv)
	if err != nil && !errors.Is(err, errNoAttr) {
		return err
	}

	content, closer, defaultMode, err := gc.Cache.FileInFileset(hash, filepath.Base(p))
	if err != nil {
		if err == cache.ErrCacheMiss {
			return err
		}
		if err == os.ErrNotExist {
			return vts.WrapWithPath(vts.WrapWithTarget(vts.WrapWithActionTarget(errors.New("file missing from build output"), resource), b), p)
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

// populateBuild implements generation of a resource target, based
// on a reference to a build target.
func populateBuild(gc GenerationContext, resource *vts.Resource, b *vts.Build) error {
	bh, err := b.RollupHash(gc.RunnerEnv, proc.EvalComputedAttribute)
	if err != nil {
		return vts.WrapWithTarget(err, b)
	}
	return writeResourceFromBuild(gc, resource, b, bh)
}
