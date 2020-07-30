// Package gen implements logic for generating on various targets.
package gen

import (
	"fmt"

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
// the given source. The provided source should have already been generated.
func PopulateResource(gc GenerationContext, resource *vts.Resource, source vts.Target) error {
	switch src := source.(type) {
	case *vts.Puesdo:
		switch src.Kind {
		case vts.FileRef:
			return populateFile(gc, resource, src)
		case vts.DebRef:
			return populateDebSource(gc, resource, src)
		}
		return fmt.Errorf("cannot generate using puesdo source %v", src.Kind)

	case *vts.Generator:
		return src.Run(resource, gc.Inputs, gc.RunnerEnv)

	case *vts.Build:
		return populateBuild(gc, resource, src)
	}

	return fmt.Errorf("cannot generate using source %T for resource %v", source, resource)
}

// Generate is called to generate a target, typically writing the output
// into the cache.
func Generate(gc GenerationContext, t vts.Target) error {
	switch t := t.(type) {
	case *vts.Resource, *vts.ResourceClass, *vts.Attr, *vts.AttrClass,
		*vts.Checker, *vts.Component, *vts.Toolchain:
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
