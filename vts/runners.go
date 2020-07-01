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
	Run(*Resource, *Checker, *RunnerEnv) error
	PopulatorsNeeded() []InfoPopulator
}

type eachAttrRunner interface {
	checkerRunner
	Run(*Attr, *Checker, *RunnerEnv) error
}

type eachComponentRunner interface {
	checkerRunner
	Run(*Component, *Checker, *RunnerEnv) error
	PopulatorsNeeded() []InfoPopulator
}

type generateRunner interface {
	Run(*Generator, *InputSet, *RunnerEnv) error
}

// InputSet describes the inputs to a generator.
type InputSet struct {
	Resource *Resource
	// Directs lists all direct inputs to the generator.
	Directs []Target
	// ClassedInputs enumerates all resources of a resource class,
	// which formed part of the input to the generator.
	ClassedResources map[*ResourceClass][]*Resource
}
