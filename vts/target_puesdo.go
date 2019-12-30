package vts

import (
	"crypto/sha256"
	"fmt"

	"go.starlark.net/starlark"
)

type PuesdoKind string

// Valid types of puesdo targets.
const (
	FileRef PuesdoKind = "file"
	DebRef  PuesdoKind = "deb"
)

// Puesdo is a special case target.
type Puesdo struct {
	Kind         PuesdoKind
	Pos          *DefPosition
	ContractPath string

	// Applicable to FileRef & DebRef targets.
	Path string

	// Applicable to DebRef targets.
	SHA256 string
	URL    string
}

func (t *Puesdo) DefinedAt() *DefPosition {
	return t.Pos
}

func (t *Puesdo) IsClassTarget() bool {
	return false
}

func (t *Puesdo) TargetType() TargetType {
	return TargetPuesdo
}

func (t *Puesdo) Validate() error {
	return nil
}

func (t *Puesdo) String() string {
	return fmt.Sprintf("puesdo<%s>", t.Kind)
}

func (t *Puesdo) Freeze() {

}

func (t *Puesdo) Truth() starlark.Bool {
	return true
}

func (t *Puesdo) Type() string {
	return "puesdo"
}

func (t *Puesdo) Hash() (uint32, error) {
	h := sha256.Sum256([]byte(fmt.Sprintf("%p", t)))
	return uint32(uint32(h[0]) + uint32(h[1])<<8 + uint32(h[2])<<16 + uint32(h[3])<<24), nil
}
