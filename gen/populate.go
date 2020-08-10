package gen

import (
	"archive/tar"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/twitchylinux/ccr/proc"
	"github.com/twitchylinux/ccr/vts"
)

func populationStrategy(gc GenerationContext, resource *vts.Resource, source vts.Target) (vts.PopulateStrategy, error) {
	// Puesdo takes precedence over declared strategies.
	if p, isPuesdo := source.(*vts.Puesdo); isPuesdo {
		switch p.Kind {
		case vts.FileRef:
			return vts.PopulateFileFirst, nil
		case vts.DebRef:
			return vts.PopulateFileMatchBasePath, nil
		}
		return 0, fmt.Errorf("cannot populate with puesdo source %v", p.Kind)
	}

	if ps := resource.Parent.Target.(*vts.ResourceClass).PopulateStrategy(); ps != 0 {
		return ps, nil
	}

	// Fallback based on source type.
	switch source.(type) {
	case *vts.Sieve, *vts.Build:
		return vts.PopulateFiles, nil
	}
	return 0, fmt.Errorf("cannot populate using source %T", source)
}

func populateFileFromCache(gc GenerationContext, outPath string, mode os.FileMode, b *vts.Build) error {
	h, err := b.RollupHash(gc.RunnerEnv, proc.EvalComputedAttribute)
	if err != nil {
		return vts.WrapWithTarget(err, b)
	}
	r, c, fMode, err := gc.Cache.FileInFileset(h, outPath)
	if err != nil {
		if err == os.ErrNotExist {
			err = errors.New("file missing from build output")
		}
		return vts.WrapWithTarget(err, b)
	}
	defer c.Close()

	if mode == 0 {
		mode = fMode
	}

	w, err := gc.RunnerEnv.FS.OpenFile(outPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, mode)
	if err != nil {
		return vts.WrapWithPath(err, outPath)
	}
	if _, err := io.Copy(w, r); err != nil {
		return vts.WrapWithPath(err, outPath)
	}
	return w.Close()
}

// PopulateResource is called to fulfill generation of a resource based on
// the given source. The provided source should have already been used as
// an argument to Generate().
func PopulateResource(gc GenerationContext, resource *vts.Resource, source vts.Target) error {
	// Special case: Generators do their own generation.
	if gen, isGen := source.(*vts.Generator); isGen {
		return gen.Run(resource, gc.Inputs, gc.RunnerEnv)
	}
	ps, err := populationStrategy(gc, resource, source)
	if err != nil {
		return err
	}

	outPath, mode, err := resourcePathMode(resource, gc.RunnerEnv)
	if err != nil {
		return err
	}

	// Fast path: Populating a single file from a build can avoid iterating
	// the fileset.
	b, isBuild := source.(*vts.Build)
	if ps == vts.PopulateFileMatchPath && isBuild {
		if err := populateFileFromCache(gc, strings.TrimPrefix(outPath, "/"), mode, b); err != nil {
			return vts.WrapWithPath(vts.WrapWithTarget(err, resource), outPath)
		}
		return nil
	}

	fsr, err := filesetForSource(gc, source)
	if err != nil {
		return err
	}
	defer fsr.Close()

	switch ps {
	case vts.PopulateFileFirst:
		err = populateFileToPath(gc.RunnerEnv.FS, fsr, outPath, mode, nil)
	case vts.PopulateFileMatchPath:
		err = populateFileToPath(gc.RunnerEnv.FS, fsr, outPath, mode, func(path string, _ *tar.Header) (bool, error) {
			return strings.TrimPrefix(path, "/") != strings.TrimPrefix(outPath, "/"), nil
		})
	case vts.PopulateFileMatchBasePath:
		err = populateFileToPath(gc.RunnerEnv.FS, fsr, outPath, mode, func(path string, _ *tar.Header) (bool, error) {
			return filepath.Base(path) != filepath.Base(outPath), nil
		})
	case vts.PopulateFiles:
		err = writeMultiFiles(gc.Cache, gc.RunnerEnv.FS, outPath, fsr)

	default:
		return fmt.Errorf("cannot generate using source %T for resource %v", source, resource)
	}

	if _, isBuildSrc := source.(*vts.Build); err == os.ErrNotExist && isBuildSrc {
		err = errors.New("file missing from build output")
	}
	if err != nil {
		err = vts.WrapWithPath(vts.WrapWithTarget(err, resource), outPath)
	}
	return err
}
