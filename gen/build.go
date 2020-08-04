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

func (rb *RunningBuild) OverlayPatchPath() string {
	return rb.env.OverlayPatchPath()
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

func (rb *RunningBuild) ExecBlocking(wd string, args []string, stdout, stderr io.Writer) (int, error) {
	id, err := rb.env.RunStreaming(wd, stdout, stderr, args...)
	if err != nil {
		return 0, err
	}
	if err := rb.env.WaitStreaming(id); err != nil {
		return 0, err
	}
	return rb.env.StreamingExitStatus(id)
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

func (rb *RunningBuild) inject(gc GenerationContext, pt vts.Target) error {
	switch t := pt.(type) {
	case *vts.Build, *vts.Sieve:
		fsr, err := filesetForSource(gc, t)
		if err != nil {
			return err
		}
		defer fsr.Close()
		return writeMultiFiles(gc.Cache, rb.fs, rb.OverlayPatchPath(), fsr)

	case *vts.Component:
		for _, d := range t.Dependencies() {
			if err := rb.inject(gc, d.Target); err != nil {
				return vts.WrapWithActionTarget(err, t)
			}
		}
		return nil

	case *vts.Resource:
		if t.Source == nil {
			return vts.WrapWithTarget(errors.New("cannot inject using virtual resource"), t)
		}
		if _, isGen := t.Source.Target.(*vts.Generator); isGen {
			return vts.WrapWithTarget(errors.New("cannot inject generator targets"), t)
		}
		gc.RunnerEnv = &vts.RunnerEnv{Dir: rb.OverlayPatchPath(), FS: osfs.New(rb.OverlayPatchPath())}
		gc.Inputs = &vts.InputSet{Resource: t}
		return PopulateResource(gc, t, t.Source.Target)
	}

	return fmt.Errorf("cannot inject target of type %T", pt)
}

func (rb *RunningBuild) patchToPath(gc GenerationContext, path string, pt vts.Target, fsr fileset) error {
	switch t := pt.(type) {
	case *vts.Build, *vts.Sieve:
		return writeMultiFiles(gc.Cache, rb.fs, filepath.Join(rb.OverlayUpperPath(), path), fsr)

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
	// cmd := exec.Command("find", rb.OverlayUpperPath())
	// cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	// cmd.Run()

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
		case vts.StepConfigure:
			if err := buildstep.RunConfigure(rb, step); err != nil {
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
		if info.IsDir() {
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
			if info.Mode()&os.ModeSymlink != 0 {
				target, err := os.Readlink(path)
				if err != nil {
					return err
				}
				if !strings.Contains(target, "..") {
					// Only do ones we can be sure are safe.
					if err := fs.AddSymlink(outPath, info, target); err != nil {
						return err
					}
				}
				return nil
			}
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

func writeMultiFiles(c *cache.Cache, fs billy.Filesystem, p string, fr fileset) error {
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

		case tar.TypeSymlink:
			if err := fs.Symlink(h.Linkname, filepath.Join(p, path)); err != nil {
				return vts.WrapWithPath(fmt.Errorf("symlink from fileset: %v", err), path)
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
	for _, inj := range b.Injections {
		if err := rb.inject(gc, inj.Target); err != nil {
			rb.Close()
			return vts.WrapWithTarget(err, b)
		}
	}

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
