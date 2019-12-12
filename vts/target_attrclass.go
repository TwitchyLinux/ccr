package vts

// AttrClass is a target representing an attribute class.
type AttrClass struct {
	Path       string
	Name       string
	Validators []TargetRef
}

func (t *AttrClass) Type() TargetType {
	return TargetAttrClass
}

func (t *AttrClass) GlobalPath() string {
	return t.Path
}

func (t *AttrClass) TargetName() string {
	return t.Name
}
