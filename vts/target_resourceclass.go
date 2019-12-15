package vts

// ResourceClass is a target representing a resource class.
type ResourceClass struct {
	Path string
	Name string

	Deps   []TargetRef
	Checks []TargetRef
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
	if err := validateDeps(t.Deps); err != nil {
		return err
	}
	return nil
}
