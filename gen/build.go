package gen

import (
	"archive/tar"
	"bytes"
	"encoding/base64"
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
	"go.starlark.net/starlark"
	"gopkg.in/src-d/go-billy.v4"
	"gopkg.in/src-d/go-billy.v4/osfs"
)

// RunningBuild represents the state for an in-progress build.
type RunningBuild struct {
	env *proc.Env

	contractDir string
	fs          billy.Filesystem
	envVars     map[string]string
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
	id, err := rb.env.RunStreaming(wd, stdout, stderr, rb.envVars, args...)
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
		if t, isGlobal := patch.Target.(vts.GlobalTarget); isGlobal {
			io.Copy(gc.Console.Stdout(), bytes.NewReader([]byte(fmt.Sprintf("-Patching \033[1;33m%s\033[0m into \033[1;33m%s\033[0m\n", t.GlobalPath(), path))))
		} else {
			io.Copy(gc.Console.Stdout(), bytes.NewReader([]byte(fmt.Sprintf("-Patching anonymous target \033[1;33m%v\033[0m into \033[1;33m%s\033[0m\n", patch, path))))
		}
		if err := rb.patch(gc, path, patch); err != nil {
			return err
		}
	}
	return nil
}

func (rb *RunningBuild) patch(gc GenerationContext, path string, patch vts.TargetRef) error {
	// Support patching in resources and components specially.
	switch t := patch.Target.(type) {
	case *vts.Component:
		for _, dep := range t.Deps {
			if err := rb.patch(gc, path, dep); err != nil {
				return err
			}
		}
		return nil
	case *vts.Resource:
		return rb.injectResourceToPath(gc, t, filepath.Join(rb.OverlayUpperPath(), path))
	}

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
	return rb.EnsurePatched(path)
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

func (rb *RunningBuild) Inject(gc GenerationContext, injections []vts.TargetRef) error {
	doneTargets := make(map[vts.Target]struct{}, 128)
	for _, inj := range injections {
		if t, isGlobal := inj.Target.(vts.GlobalTarget); isGlobal {
			io.Copy(gc.Console.Stdout(), bytes.NewReader([]byte(fmt.Sprintf("-Injecting \033[1;33m%s\033[0m\n", t.GlobalPath()))))
		} else {
			io.Copy(gc.Console.Stdout(), bytes.NewReader([]byte(fmt.Sprintf("-Injecting anonymous target \033[1;33m%v\033[0m\n", inj))))
		}

		if err := rb.inject(gc, inj.Target, doneTargets); err != nil {
			return err
		}
	}
	return nil
}

func (rb *RunningBuild) inject(gc GenerationContext, pt vts.Target, doneTargets map[vts.Target]struct{}) error {
	if _, isComponent := pt.(*vts.Component); !isComponent {
		if _, done := doneTargets[pt]; done {
			return nil
		}
		doneTargets[pt] = struct{}{}
	}

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
			if err := rb.inject(gc, d.Target, doneTargets); err != nil {
				return vts.WrapWithActionTarget(err, t)
			}
		}
		return nil

	case *vts.Resource:
		return rb.injectResourceToPath(gc, t, rb.OverlayPatchPath())
	}

	return fmt.Errorf("cannot inject target of type %T", pt)
}

var permittedInjectGenerators = map[*vts.Generator]struct{}{
	common.SymlinkGenerator:        struct{}{},
	common.DirGenerator:            struct{}{},
	common.SysLibUnionLinkerscript: struct{}{},
}

func (rb *RunningBuild) injectResourceToPath(gc GenerationContext, t *vts.Resource, path string) error {
	if t.Source == nil {
		return vts.WrapWithTarget(errors.New("cannot inject using virtual resource"), t)
	}
	gc.RunnerEnv = &vts.RunnerEnv{Dir: path, FS: osfs.New(path)}
	gc.Inputs = &vts.InputSet{Resource: t}
	if gen, isGen := t.Source.Target.(*vts.Generator); isGen {
		if _, permitted := permittedInjectGenerators[gen]; !permitted {
			return vts.WrapWithTarget(errors.New("cannot inject generator targets"), t)
		}
	}
	return PopulateResource(gc, t, t.Source.Target)
}

func (rb *RunningBuild) Generate(c *cache.Cache, o, e io.Writer) error {
	// cmd := exec.Command("find", rb.OverlayUpperPath())
	// cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	// cmd.Run()

	for i, step := range rb.steps {
		switch step.Kind {
		case vts.StepUnpackGz, vts.StepUnpackXz, vts.StepUnpackBz2:
			if err := buildstep.RunUnpack(c, rb, step); err != nil {
				return fmt.Errorf("step %d (%s) failed: %v", i+1, step.Kind, err)
			}
			if err := rb.EnsurePatched(step.ToPath); err != nil {
				return fmt.Errorf("step %d (%s) failed wiring into filesystem: %v", i+1, step.Kind, err)
			}
		case vts.StepShellCmd:
			if err := buildstep.RunShellCmd(rb, step, o, e); err != nil {
				return fmt.Errorf("step %d (%s) failed: %v", i+1, step.Kind, err)
			}
		case vts.StepConfigure:
			if err := buildstep.RunConfigure(rb, step, o, e); err != nil {
				return fmt.Errorf("step %d (%s) failed: %v", i+1, step.Kind, err)
			}
		case vts.StepPatch:
			if err := buildstep.RunPatch(rb, step); err != nil {
				return fmt.Errorf("step %d (%s) failed: %v", i+1, step.Kind, err)
			}
		case vts.StepWrite:
			if err := buildstep.RunWrite(rb, step); err != nil {
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

func determinePrefix(prefix string) string {
	if i := strings.LastIndex(prefix, ":"); i > 0 && !strings.HasSuffix(prefix, ":build") {
		prefix = prefix[i+1:]
	} else {
		prefix = strings.Split(prefix, ":")[0]
		if base := filepath.Base(prefix); base != "build" {
			prefix = base
		}
	}
	return prefix
}

// generateBuild executes a build if the result is not already cached.
func generateBuild(gc GenerationContext, b *vts.Build) error {
	bh, err := b.RollupHash(gc.RunnerEnv, proc.EvalComputedAttribute)
	if err != nil {
		return vts.WrapWithTarget(err, b)
	}
	// See if its already cached.
	isCached, err := gc.Cache.IsHashCached(bh)
	if err != nil {
		return err
	}
	if isCached {
		return nil
	}

	envVars := make(map[string]string, len(b.Env))
	for k, v := range b.Env {
		if ss, ok := v.(starlark.String); ok {
			envVars[k] = string(ss)
		} else {
			envVars[k] = v.String()
		}
	}

	prefix := determinePrefix(b.GlobalPath())
	msg := fmt.Sprintf("Starting \033[1;36m%s\033[0m of \033[1;33m%s\033[0m\n", "build", b.GlobalPath())
	gc.Console = gc.Console.Operation(base64.RawURLEncoding.EncodeToString(bh)[:36], msg, prefix)
	defer gc.Console.Done()

	// If we got this far, the build output is not cached, we need to complete the build manually.
	env, err := proc.NewEnv(false)
	if err != nil {
		return vts.WrapWithTarget(fmt.Errorf("creating build environment: %v", err), b)
	}
	rb := RunningBuild{
		env:         env,
		steps:       b.Steps,
		fs:          osfs.New("/"),
		envVars:     envVars,
		contractDir: b.ContractDir,
	}
	if err := rb.Inject(gc, b.Injections); err != nil {
		rb.Close()
		return vts.WrapWithTarget(fmt.Errorf("failed to apply injections: %v", err), b)
	}

	if err := rb.Patch(gc, b.PatchIns); err != nil {
		rb.Close()
		return vts.WrapWithTarget(fmt.Errorf("failed to apply patch-ins: %v", err), b)
	}
	if err := rb.Generate(gc.Cache, gc.Console.Stdout(), gc.Console.Stderr()); err != nil {
		rb.Close()
		return vts.WrapWithTarget(fmt.Errorf("build failed: %v", err), b)
	}
	if err := rb.WriteToCache(gc.Cache, b, bh); err != nil {
		rb.Close()
		return vts.WrapWithTarget(fmt.Errorf("gathering output: %v", err), b)
	}
	return rb.Close()
}
