package vts

import (
	"crypto/sha256"
	"fmt"

	"go.starlark.net/starlark"
)

// Attr is a target representing an attribute.
type Attr struct {
	Path        string
	Name        string
	ParentClass TargetRef

	Value starlark.Value
}

func (t *Attr) TargetType() TargetType {
	return TargetAttr
}

func (t *Attr) GlobalPath() string {
	return t.Path
}

func (t *Attr) TargetName() string {
	return t.Name
}

func (t *Attr) String() string {
	return fmt.Sprintf("attr<%s>", t.Name)
}

func (t *Attr) Freeze() {

}

func (t *Attr) Truth() starlark.Bool {
	return true
}

func (t *Attr) Type() string {
	return "attr"
}

func (t *Attr) Hash() (uint32, error) {
	h := sha256.Sum256([]byte(fmt.Sprintf("%p", t)))
	return uint32(uint32(h[0]) + uint32(h[1])<<8 + uint32(h[2])<<16 + uint32(h[3])<<24), nil
}
