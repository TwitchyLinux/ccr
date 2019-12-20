package vts

import (
	"crypto/sha256"
	"fmt"

	"go.starlark.net/starlark"
)

// Generator is a target representing a generator.
type Generator struct {
	Path string
	Name string
	Pos  *DefPosition
}

func (t *Generator) DefinedAt() *DefPosition {
	return t.Pos
}

func (t *Generator) IsClassTarget() bool {
	return false
}

func (t *Generator) TargetType() TargetType {
	return TargetGenerator
}

func (t *Generator) GlobalPath() string {
	return t.Path
}

func (t *Generator) TargetName() string {
	return t.Name
}

func (t *Generator) Validate() error {
	return nil
}

func (t *Generator) String() string {
	return fmt.Sprintf("generator<%s>", t.Name)
}

func (t *Generator) Freeze() {

}

func (t *Generator) Truth() starlark.Bool {
	return true
}

func (t *Generator) Type() string {
	return "generator"
}

func (t *Generator) Hash() (uint32, error) {
	h := sha256.Sum256([]byte(fmt.Sprintf("%p", t)))
	return uint32(uint32(h[0]) + uint32(h[1])<<8 + uint32(h[2])<<16 + uint32(h[3])<<24), nil
}
