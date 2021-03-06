package vts

import (
	"crypto/sha256"
	"errors"
	"fmt"

	"go.starlark.net/starlark"
)

// ComputedValue is an anonymous target representing a runtime computation.
type ComputedValue struct {
	Pos          *DefPosition
	ContractDir  string
	ContractPath string

	Filename     string
	Func         string
	InlineScript []byte
	ReadWrite    bool
}

func (t *ComputedValue) DefinedAt() *DefPosition {
	return t.Pos
}

func (t *ComputedValue) IsClassTarget() bool {
	return false
}

func (t *ComputedValue) Validate() error {
	if t.Filename == "" && len(t.InlineScript) == 0 {
		return errors.New("filename or code must be specified")
	}
	if t.Func == "" && len(t.InlineScript) == 0 {
		return errors.New("function or code must be specified")
	}
	return nil
}

func (t *ComputedValue) String() string {
	if len(t.InlineScript) > 0 {
		hsh := sha256.Sum256(t.InlineScript)
		return fmt.Sprintf("computed_value<0x%X>", hsh[:4])
	}
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
