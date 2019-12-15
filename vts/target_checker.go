package vts

import (
	"crypto/sha256"
	"fmt"

	"go.starlark.net/starlark"
)

type CheckerKind string

// Valid checker kinds.
const (
	ChkKindEachResource CheckerKind = "each_resource"
)

// Checker is a target describing a check on a target or the system.
type Checker struct {
	Path string
	Name string
	Kind CheckerKind
}

func (t *Checker) TargetType() TargetType {
	return TargetChecker
}

func (t *Checker) GlobalPath() string {
	return t.Path
}

func (t *Checker) TargetName() string {
	return t.Name
}

func (t *Checker) Validate() error {
	switch t.Kind {
	case ChkKindEachResource:
	default:
		return fmt.Errorf("invalid checker kind: %q", t.Kind)
	}
	return nil
}

func (t *Checker) String() string {
	return fmt.Sprintf("checker<%s, %s>", t.Name, t.Kind)
}

func (t *Checker) Freeze() {

}

func (t *Checker) Truth() starlark.Bool {
	return true
}

func (t *Checker) Type() string {
	return "checker"
}

func (t *Checker) Hash() (uint32, error) {
	h := sha256.Sum256([]byte(fmt.Sprintf("%p", t)))
	return uint32(uint32(h[0]) + uint32(h[1])<<8 + uint32(h[2])<<16 + uint32(h[3])<<24), nil
}
