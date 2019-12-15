package vts

// Component is a target representing a component.
type Component struct {
	Path string
	Name string

	Details []TargetRef
	Deps    []TargetRef
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
