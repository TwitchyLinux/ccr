// Package gen implements logic for generating on various targets.
package gen

import (
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
