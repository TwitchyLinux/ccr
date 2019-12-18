package vts

// CheckerOpts describes state and configuration information used by checkers.
type CheckerOpts struct {
	Dir string
}

type checkerRunner interface {
	Kind() CheckerKind
}

type eachResourceRunner interface {
	checkerRunner
	Run(*Resource, *CheckerOpts) error
}

type eachAttrRunner interface {
	checkerRunner
	Run(*Attr, *CheckerOpts) error
}
