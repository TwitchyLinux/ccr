package runners

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/twitchylinux/ccr/proc"
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

func (*pathValidRunner) Run(attr *vts.Attr, chkr *vts.Checker, opts *vts.RunnerEnv) error {
	v, err := attr.Value(chkr, opts, proc.EvalComputedAttribute)
	if err != nil {
		return err
	}
	path := v.String()
	if p, ok := v.(starlark.String); ok {
		path = string(p)
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

func (e *enumValueRunner) Run(attr *vts.Attr, chkr *vts.Checker, opts *vts.RunnerEnv) error {
	v, err := attr.Value(chkr, opts, proc.EvalComputedAttribute)
	if err != nil {
		return err
	}
	s := v.String()
	if ss, ok := v.(starlark.String); ok {
		s = string(ss)
	}
	if _, ok := e.allowedValues[s]; !ok {
		return fmt.Errorf("invalid value: %q", s)
	}
	return nil
}

// OctalCheckValid returns a runner that can check attrs
// are valid octal strings.
func OctalCheckValid() *modeValidRunner {
	return &modeValidRunner{}
}

type modeValidRunner struct{}

func (*modeValidRunner) Kind() vts.CheckerKind { return vts.ChkKindEachAttr }

func (*modeValidRunner) String() string { return "attr.octal_valid" }

func (*modeValidRunner) Freeze() {}

func (*modeValidRunner) Truth() starlark.Bool { return true }

func (*modeValidRunner) Type() string { return "runner" }

func (t *modeValidRunner) Hash() (uint32, error) {
	h := sha256.Sum256([]byte(fmt.Sprintf("%p", t)))
	return uint32(uint32(h[0]) + uint32(h[1])<<8 + uint32(h[2])<<16 + uint32(h[3])<<24), nil
}

func (*modeValidRunner) Run(attr *vts.Attr, chkr *vts.Checker, opts *vts.RunnerEnv) error {
	v, err := attr.Value(chkr, opts, proc.EvalComputedAttribute)
	if err != nil {
		return err
	}
	mode := v.String()
	if m, ok := v.(starlark.String); ok {
		mode = string(m)
	}
	if mode == "" {
		return errors.New("mode was empty")
	}
	for i := range mode {
		if !strings.ContainsAny(string(mode[i]), "01234567") {
			return fmt.Errorf("invalid mode: char %q is not a valid octal character", mode[i])
		}
	}
	if _, err := strconv.ParseInt(mode, 8, 64); err != nil {
		return err
	}
	return nil
}

// BooleanCheckValid returns a runner that can check attrs
// are valid booleans.
func BooleanCheckValid() *boolCheckValid {
	return &boolCheckValid{}
}

type boolCheckValid struct{}

func (*boolCheckValid) Kind() vts.CheckerKind { return vts.ChkKindEachAttr }

func (*boolCheckValid) String() string { return "attr.bool_valid" }

func (*boolCheckValid) Freeze() {}

func (*boolCheckValid) Truth() starlark.Bool { return true }

func (*boolCheckValid) Type() string { return "runner" }

func (t *boolCheckValid) Hash() (uint32, error) {
	h := sha256.Sum256([]byte(fmt.Sprintf("%p", t)))
	return uint32(uint32(h[0]) + uint32(h[1])<<8 + uint32(h[2])<<16 + uint32(h[3])<<24), nil
}

func (*boolCheckValid) Run(attr *vts.Attr, chkr *vts.Checker, opts *vts.RunnerEnv) error {
	v, err := attr.Value(chkr, opts, proc.EvalComputedAttribute)
	if err != nil {
		return err
	}
	if _, isBool := v.(starlark.Bool); !isBool {
		return fmt.Errorf("attr is not a boolean: got type %T", v)
	}
	return nil
}
