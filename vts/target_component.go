package vts

// Component is a target representing a component.
type Component struct {
	Path string
	Name string
	Pos  *DefPosition

	Details []TargetRef
	Deps    []TargetRef
	Checks  []TargetRef

	Info RuntimeInfo
}

func (t *Component) DefinedAt() *DefPosition {
	return t.Pos
}

func (t *Component) IsClassTarget() bool {
	return false
}

func (t *Component) TargetType() TargetType {
	return TargetComponent
}

func (t *Component) GlobalPath() string {
	return t.Path
}

func (t *Component) TargetName() string {
	return t.Name
}

func (t *Component) Dependencies() []TargetRef {
	return t.Deps
}

func (t *Component) Checkers() []TargetRef {
	return t.Checks
}

func (t *Component) Attributes() []TargetRef {
	return t.Details
}

func (t *Component) RuntimeInfo() *RuntimeInfo {
	return &t.Info
}

func (t *Component) Validate() error {
	if err := validateDetails(t.Details); err != nil {
		return err
	}
	if err := validateDeps(t.Deps); err != nil {
		return err
	}
	return nil
}
