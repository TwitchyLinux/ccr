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
