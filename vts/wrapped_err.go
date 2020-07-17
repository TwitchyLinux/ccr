package vts

// WrappedErr provides additional context to an error which occurred when
// working with virtual targets.
type WrappedErr struct {
	ComputedValue *ComputedValue
	Target        Target
	ActionTarget  Target // Generator or checker that lead to the error
	TargetChain   []Target
	Path          string
	Pos           *DefPosition
	Err           error
}

// Error implements the error interface.
func (e WrappedErr) Error() string {
	return e.Err.Error()
}

func WrapWithComputedValue(err error, c *ComputedValue) WrappedErr {
	if we, ok := err.(WrappedErr); ok {
		we.ComputedValue = c
		return we
	}
	return WrappedErr{Err: err, ComputedValue: c}
}

// WrapWithPath wraps an error with path information.
func WrapWithPath(err error, path string) WrappedErr {
	if we, ok := err.(WrappedErr); ok {
		we.Path = path
		return we
	}
	return WrappedErr{Err: err, Path: path}
}

// WrapWithTarget wraps an error with target information.
func WrapWithTarget(err error, t Target) WrappedErr {
	if we, ok := err.(WrappedErr); ok {
		if we.Target == nil {
			we.Target = t
		} else if we.Target == t {
			// Already set.
		} else {
			we.TargetChain = append(we.TargetChain, t)
		}
		return we
	}
	return WrappedErr{Err: err, Target: t}
}

// WrapWithActionTarget wraps an error with action target information.
func WrapWithActionTarget(err error, t Target) WrappedErr {
	if we, ok := err.(WrappedErr); ok {
		if we.ActionTarget == nil {
			we.ActionTarget = t
		}
		return we
	}
	return WrappedErr{Err: err, ActionTarget: t}
}

// WrapWithPosition wraps an error with declaration position information.
func WrapWithPosition(err error, pos *DefPosition) WrappedErr {
	if we, ok := err.(WrappedErr); ok {
		we.Pos = pos
		return we
	}
	return WrappedErr{Err: err, Pos: pos}
}
