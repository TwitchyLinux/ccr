package ccbuild

import (
	"crypto/sha256"
	"errors"
	"fmt"

	"github.com/twitchylinux/ccr/vts"
	"go.starlark.net/starlark"
	"go.starlark.net/syntax"
)

// RefComparisonConstraint implements a comparison constraint on a target
// and one of its attributes.
type RefComparisonConstraint struct {
	Target       starlark.Value
	AttrClass    *vts.AttrClass
	CompareValue starlark.Value
	Op           syntax.Token
}

func (c *RefComparisonConstraint) String() string {
	return c.Type()
}

// Type implements starlark.Value.
func (c *RefComparisonConstraint) Type() string {
	return "target_constraint<cmp>"
}

// Freeze implements starlark.Value.
func (c *RefComparisonConstraint) Freeze() {
}

// Truth implements starlark.Value.
func (c *RefComparisonConstraint) Truth() starlark.Bool {
	return true
}

// Hash implements starlark.Value.
func (c *RefComparisonConstraint) Hash() (uint32, error) {
	h := sha256.Sum256([]byte(c.String()))
	return uint32(uint32(h[0]) + uint32(h[1])<<8 + uint32(h[2])<<16 + uint32(h[3])<<24), nil
}

func (c *RefComparisonConstraint) Binary(op syntax.Token, y starlark.Value, side starlark.Side) (starlark.Value, error) {
	c.Op = op
	switch op {
	case syntax.GTGT:
	default:
		return nil, fmt.Errorf("cannot handle constraint with op %q", op.String())
	}
	if side != starlark.Right {
		return nil, fmt.Errorf("invalid constraint: must be specified to the right")
	}
	c.Target = y
	return c, nil
}

func (c *RefComparisonConstraint) Name() string { return c.Type() }

func (c *RefComparisonConstraint) CallInternal(thread *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	return nil, errors.New("not implemented")
}
