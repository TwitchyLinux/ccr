package ccbuild

import (
	"crypto/sha256"
	"fmt"

	semver "github.com/blang/semver/v4"
	"github.com/twitchylinux/ccr/vts"
	"github.com/twitchylinux/ccr/vts/common"
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
	case syntax.GTGT, syntax.LTLT:
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

func (c *RefComparisonConstraint) Check(env *vts.RunnerEnv, lhs starlark.Value) error {
	l, ok := lhs.(starlark.String)
	if !ok {
		return fmt.Errorf("lhs was of type %T, want string", lhs)
	}
	r, ok := c.CompareValue.(starlark.String)
	if !ok {
		return fmt.Errorf("rhs was of type %T, want string", c.CompareValue)
	}

	switch c.AttrClass {
	case common.SemverClass:
		lv, err := semver.ParseTolerant(string(l))
		if err != nil {
			return fmt.Errorf("lhs: %v", err)
		}
		rv, err := semver.ParseTolerant(string(r))
		if err != nil {
			return fmt.Errorf("rhs: %v", err)
		}

		switch c.Op {
		case syntax.GTGT:
			if lv.GT(rv) {
				return nil
			}
		case syntax.LTLT:
			if lv.LT(rv) {
				return nil
			}
		}
		return vts.FailingConstraintInfo{
			Lhs:  string(l),
			Rhs:  fmt.Sprintf("semver(%q)", string(r)),
			Op:   c.Op.String(),
			Kind: "semver",
		}
	}

	return fmt.Errorf("attr class %q not supported", c.AttrClass.GlobalPath())
}
