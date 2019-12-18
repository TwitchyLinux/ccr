package runners

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"strings"

	"github.com/twitchylinux/ccr/vts"
	"go.starlark.net/starlark"
)

// PathCheckValid returns a runner that can check attrs
// are valid paths.
func PathCheckValid() *pathValidRunner {
	return &pathValidRunner{}
}

type pathValidRunner struct{}

func (*pathValidRunner) Kind() vts.CheckerKind { return vts.ChkKindEachAttr }

func (*pathValidRunner) String() string { return "attr.path_valid" }

func (*pathValidRunner) Freeze() {}

func (*pathValidRunner) Truth() starlark.Bool { return true }

func (*pathValidRunner) Type() string { return "runner" }

func (t *pathValidRunner) Hash() (uint32, error) {
	h := sha256.Sum256([]byte(fmt.Sprintf("%p", t)))
	return uint32(uint32(h[0]) + uint32(h[1])<<8 + uint32(h[2])<<16 + uint32(h[3])<<24), nil
}

func (*pathValidRunner) Run(attr *vts.Attr, opts *vts.CheckerOpts) error {
	path := attr.Value.String()
	if s, ok := attr.Value.(starlark.String); ok {
		path = string(s)
	}
	if path == "" {
		return errors.New("path was empty")
	}
	if strings.ContainsAny(path, "\x00:<>") {
		return errors.New("path contains illegal characters")
	}
	return nil
}

// EnumCheckValid returns a runner which validates all values are
// one of the provided allowed values.
func EnumCheckValid(allowedValues []string) *enumValueRunner {
	e := enumValueRunner{allowedValues: make(map[string]struct{}, len(allowedValues))}
	for _, v := range allowedValues {
		e.allowedValues[v] = struct{}{}
	}
	return &e
}

type enumValueRunner struct {
	allowedValues map[string]struct{}
}

func (*enumValueRunner) Kind() vts.CheckerKind { return vts.ChkKindEachAttr }

func (*enumValueRunner) String() string { return "attr.enum_valid" }

func (*enumValueRunner) Freeze() {}

func (*enumValueRunner) Truth() starlark.Bool { return true }

func (*enumValueRunner) Type() string { return "runner" }

func (t *enumValueRunner) Hash() (uint32, error) {
	h := sha256.Sum256([]byte(fmt.Sprintf("%p", t)))
	return uint32(uint32(h[0]) + uint32(h[1])<<8 + uint32(h[2])<<16 + uint32(h[3])<<24), nil
}

func (e *enumValueRunner) Run(attr *vts.Attr, opts *vts.CheckerOpts) error {
	v := attr.Value.String()
	if s, ok := attr.Value.(starlark.String); ok {
		v = string(s)
	}
	if _, ok := e.allowedValues[v]; !ok {
		return fmt.Errorf("invalid value: %q", v)
	}
	return nil
}
