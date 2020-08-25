// Package gen implements logic for generating targets, and populating
// resources from a source target.
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
	Console   vts.Console
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
