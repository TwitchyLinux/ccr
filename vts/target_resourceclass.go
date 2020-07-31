package vts

import "fmt"

// PopulateStrategy describes how files should be read from the source
// and written into the output filesystem, when a resource is being
// generated.
type PopulateStrategy uint8

const (
	// PopulateFileMatchPath indicates a single file should be populated from
	// the source, where the source path and resource path are an exact match.
	PopulateFileMatchPath PopulateStrategy = iota + 1
	// PopulateFileFirst indicates a single file should be populated from the
	// source, where the first file in the source is used.
	PopulateFileFirst
	// PopulateFileMatchBasePath indicates a single file should be populated
	// from the source, where the filename of the resource path matches the
	// path of the file in the source.
	PopulateFileMatchBasePath
	// PopulateFiles indicates all files from the source should be populated,
	// with file paths from the source being joined with the resource path
	// to determine the path the files should be written.
	PopulateFiles
)

// Unary returns true if the population strategy is about emitting a single
// file.
func (s PopulateStrategy) Unary() bool {
	switch s {
	case PopulateFiles:
		return false
	}
	return true
}

// ResourceClass is a target representing a resource class.
type ResourceClass struct {
	Path string
	Name string
	Pos  *DefPosition

	PopStrategy PopulateStrategy
	Deps        []TargetRef
	Checks      []TargetRef
}

func (t *ResourceClass) DefinedAt() *DefPosition {
	return t.Pos
}

func (t *ResourceClass) IsClassTarget() bool {
	return true
}

func (t *ResourceClass) TargetType() TargetType {
	return TargetResourceClass
}

func (t *ResourceClass) GlobalPath() string {
	return t.Path
}

func (t *ResourceClass) TargetName() string {
	return t.Name
}

func (t *ResourceClass) Dependencies() []TargetRef {
	return t.Deps
}

func (t *ResourceClass) Checkers() []TargetRef {
	return t.Checks
}

func (t *ResourceClass) Validate() error {
	if err := validateDeps(t.Deps, false); err != nil {
		return err
	}
	return nil
}

func (t *ResourceClass) PopulateStrategy() PopulateStrategy {
	return t.PopStrategy
}

// RunCheckers runs checkers on a resource, in the context of this
// resource class.
func (t *ResourceClass) RunCheckers(r *Resource, opts *RunnerEnv) error {
	for i, c := range t.Checks {
		if c.Target == nil {
			return fmt.Errorf("check[%d] is not resolved: %q", i, c.Path)
		}
		if err := c.Target.(*Checker).RunResource(r, opts); err != nil {
			return err
		}
	}
	return nil
}
