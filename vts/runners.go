package vts

import (
	"io"

	billy "gopkg.in/src-d/go-billy.v4"
)

// RunnerEnv describes state and configuration information used by runners.
type RunnerEnv struct {
	Dir      string
	FS       billy.Filesystem
	Universe UniverseResolver
}

type Console interface {
	Operation(key, msg, prefix string) Console
	Done() error
	Stdout() io.Writer
	Stderr() io.Writer
}

// UniverseResolver allows resolution of targets within the universe.
type UniverseResolver interface {
	FindByPath(path string, env *RunnerEnv) (Target, error)
	AllTargets() []GlobalTarget
	GetData(key string) (interface{}, bool)
	SetData(key string, data interface{})
	Inject(t Target) (Target, error)
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

type globalRunner interface {
	checkerRunner
	Run(*Checker, *RunnerEnv) error
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
