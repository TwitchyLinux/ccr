package vts

// AttrClass is a target representing an attribute class.
type AttrClass struct {
	Path   string
	Name   string
	Checks []TargetRef
}

func (t *AttrClass) TargetType() TargetType {
	return TargetAttrClass
}

func (t *AttrClass) GlobalPath() string {
	return t.Path
}

func (t *AttrClass) TargetName() string {
	return t.Name
}

func (t *AttrClass) Checkers() []TargetRef {
	return t.Checks
}
