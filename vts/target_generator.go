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

	Inputs []TargetRef
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
	for i, dep := range t.Inputs {
		_, component := dep.Target.(*Component)
		_, resource := dep.Target.(*Resource)
		_, resourceClass := dep.Target.(*ResourceClass)
		if !component && !resource && !resourceClass {
			return fmt.Errorf("deps[%d] is type %T, but must be resource/resource-class/component", i, dep.Target)
		}
	}
	return nil
}

func (t *Generator) NeedInputs() []TargetRef {
	return t.Inputs
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
