// Package gen implements logic for generating on various targets.
package gen

import (
	"github.com/twitchylinux/ccr/ccr/deb"
	"github.com/twitchylinux/ccr/vts"
)

// GenerationContext encodes additional information that may be needed when
// generating a resource.
type GenerationContext struct {
	RunnerEnv *vts.RunnerEnv
	Cache     Cache
	Inputs    *vts.InputSet
}

// Cache describes a caching layer to be used opportunistically when
// downloading or looking up larger objects or inputs.
type Cache interface {
	NamePath(string) string
	SHA256Path(string) string
	BySHA256(string) (deb.ReadSeekCloser, error)

	PutObj(sha256 string, v interface{})
	GetObj(sha256 string) (value interface{}, ok bool)
}
