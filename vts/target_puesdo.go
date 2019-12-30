package vts

import (
	"crypto/sha256"
	"errors"
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
	Name         string
	TargetPath   string
	ContractPath string

	// Applicable to FileRef & DebRef targets.
	Path string

	// Applicable to DebRef targets.
	SHA256 string
	URL    string

	Details []TargetRef
}

func (t *Puesdo) DefinedAt() *DefPosition {
	return t.Pos
}

func (t *Puesdo) IsClassTarget() bool {
	return false
}

func (t *Puesdo) GlobalPath() string {
	return t.TargetPath
}

func (t *Puesdo) TargetName() string {
	return t.Name
}

func (t *Puesdo) TargetType() TargetType {
	return TargetPuesdo
}

func (t *Puesdo) Attributes() []TargetRef {
	return t.Details
}

func (t *Puesdo) Validate() error {
	switch t.Kind {
	case DebRef:
		if t.Path == "" {
			return errors.New("deb source cannot have empty path")
		}
		if t.URL == "" {
			return errors.New("deb source must specify a URL")
		}
		if t.SHA256 == "" {
			return errors.New("deb source must specify a sha256 hash")
		}
		return nil
	case FileRef:
		if t.Path == "" {
			return errors.New("file source cannot have empty path")
		}
		return nil
	default:
		return fmt.Errorf("unrecognized puesdotarget: %v", t.Kind)
	}
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
