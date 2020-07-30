package match

import (
	"crypto/sha256"
	"fmt"
	"strings"

	"go.starlark.net/starlark"
)

// OutputMapper types map an input filename into an output filename.
type OutputMapper interface {
	Map(string) string
}

// StripPrefixOutputMapper implements BuildOutputMapper by computing the
// output path based on the existing filename, but with a fixed
// prefix removed.
type StripPrefixOutputMapper struct {
	Prefix string
}

func (c *StripPrefixOutputMapper) String() string {
	return c.Type()
}

// Type implements starlark.Value.
func (c *StripPrefixOutputMapper) Type() string {
	return fmt.Sprintf("strip_prefix_mapper<%q>", c.Prefix)
}

// Freeze implements starlark.Value.
func (c *StripPrefixOutputMapper) Freeze() {
}

// Truth implements starlark.Value.
func (c *StripPrefixOutputMapper) Truth() starlark.Bool {
	return true
}

// Hash implements starlark.Value.
func (c *StripPrefixOutputMapper) Hash() (uint32, error) {
	h := sha256.Sum256([]byte(c.String()))
	return uint32(uint32(h[0]) + uint32(h[1])<<8 + uint32(h[2])<<16 + uint32(h[3])<<24), nil
}

func (c *StripPrefixOutputMapper) Map(originalPath string) string {
	return strings.TrimPrefix(originalPath, c.Prefix)
}

// LiteralOutputMapper implements BuildOutputMapper, for hardcoded
// output paths.
type LiteralOutputMapper string

func (m LiteralOutputMapper) Map(_ string) string {
	return string(m)
}
