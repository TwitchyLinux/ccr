package vts

// Resource is a target representing a resource.
type Resource struct {
	Path    string
	Name    string
	Parent  TargetRef
	Details []TargetRef
	Deps    []TargetRef
}

func (t *Resource) TargetType() TargetType {
	return TargetResource
}

func (t *Resource) GlobalPath() string {
	return t.Path
}

func (t *Resource) TargetName() string {
	return t.Name
}
