// Package gen implements logic for generating on various targets.
package gen

import (
	"archive/tar"
	"fmt"
	"path/filepath"

	"github.com/twitchylinux/ccr/cache"
	"github.com/twitchylinux/ccr/vts"
)

// GenerationContext encodes additional information that may be needed when
// generating a resource.
type GenerationContext struct {
	RunnerEnv *vts.RunnerEnv
	Cache     *cache.Cache
	Inputs    *vts.InputSet
}

// PopulateResource is called to fulfill generation of a resource based on
// the given source. The provided source should have already been used as
// an argument to Generate().
func PopulateResource(gc GenerationContext, resource *vts.Resource, source vts.Target) error {
	// Special case: Generators do their own generation.
	if gen, isGen := source.(*vts.Generator); isGen {
		return gen.Run(resource, gc.Inputs, gc.RunnerEnv)
	}
	fsr, err := filesetForSource(gc, source)
	if err != nil {
		return err
	}
	defer fsr.Close()

	switch src := source.(type) {
	case *vts.Puesdo:
		outPath, mode, err := resourcePathMode(resource, gc.RunnerEnv)
		if err != nil {
			return err
		}

		switch src.Kind {
		case vts.FileRef:
			return populateFileToPath(gc.RunnerEnv.FS, fsr, outPath, mode, nil)
		case vts.DebRef:
			return populateFileToPath(gc.RunnerEnv.FS, fsr, outPath, mode, func(path string, _ *tar.Header) (bool, error) {
				return path != outPath, nil
			})
		}
		return fmt.Errorf("cannot generate using puesdo source %v", src.Kind)

	case *vts.Sieve:
		outPath, mode, err := resourcePathMode(resource, gc.RunnerEnv)
		if err != nil {
			return err
		}
		return populateFileToPath(gc.RunnerEnv.FS, fsr, outPath, mode, func(path string, _ *tar.Header) (bool, error) {
			return path != filepath.Base(outPath), nil
		})

	case *vts.Build:
		return writeResourceFromBuild(gc, resource, fsr)
	}

	return fmt.Errorf("cannot generate using source %T for resource %v", source, resource)
}

// Generate is called to generate a target, typically writing the output
// into the cache.
func Generate(gc GenerationContext, t vts.Target) error {
	switch t := t.(type) {
	case *vts.Resource, *vts.ResourceClass, *vts.Attr, *vts.AttrClass,
		*vts.Checker, *vts.Component, *vts.Toolchain, *vts.Sieve:
		return nil // Targets dont require direct generation.
	case *vts.Generator:
		return nil // Generators are always fulfilled with their source.
	case *vts.Puesdo:
		return nil // For now these always run through PopulateResource().
	case *vts.Build:
		return generateBuild(gc, t)
	}
	return fmt.Errorf("generate: unknown target type %T", t)
}
