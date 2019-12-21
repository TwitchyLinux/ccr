package vts

import (
	billy "gopkg.in/src-d/go-billy.v4"
)

// RunnerEnv describes state and configuration information used by runners.
type RunnerEnv struct {
	Dir string
	FS  billy.Filesystem
}

type checkerRunner interface {
	Kind() CheckerKind
}

type eachResourceRunner interface {
	checkerRunner
	Run(*Resource, *RunnerEnv) error
}

type eachAttrRunner interface {
	checkerRunner
	Run(*Attr, *RunnerEnv) error
}

type eachComponentRunner interface {
	checkerRunner
	Run(*Component, *RunnerEnv) error
}
