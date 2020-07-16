package vts

import (
	"crypto/sha256"
	"errors"
	"fmt"

	"go.starlark.net/starlark"
)

// ComputedValue is an anonymous target representing a runtime computation.
type ComputedValue struct {
	Pos *DefPosition

	Filename string
	Func     string
}

func (t *ComputedValue) DefinedAt() *DefPosition {
	return t.Pos
}

func (t *ComputedValue) IsClassTarget() bool {
	return false
}

func (t *ComputedValue) Validate() error {
	if t.Filename == "" {
		return errors.New("filename must be specified")
	}
	if t.Func == "" {
		return errors.New("function must be specified")
	}
	return nil
}

func (t *ComputedValue) String() string {
	return fmt.Sprintf("computed_value<%s, %s>", t.Filename, t.Func)
}

func (t *ComputedValue) Freeze() {

}

func (t *ComputedValue) Truth() starlark.Bool {
	return true
}

func (t *ComputedValue) Type() string {
	return "computed_value"
}

func (t *ComputedValue) Hash() (uint32, error) {
	h := sha256.Sum256([]byte(fmt.Sprintf("%p", t)))
	return uint32(uint32(h[0]) + uint32(h[1])<<8 + uint32(h[2])<<16 + uint32(h[3])<<24), nil
}
