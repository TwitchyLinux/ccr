package vts

import (
	billy "gopkg.in/src-d/go-billy.v4"
)

// CheckerOpts describes state and configuration information used by checkers.
type CheckerOpts struct {
	Dir string
	FS  billy.Filesystem
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

type eachComponentRunner interface {
	checkerRunner
	Run(*Component, *CheckerOpts) error
}
