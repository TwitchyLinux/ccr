package vts

// Attr is a target representing an attribute.
type Attr struct {
	Path        string
	Name        string
	ParentClass TargetRef
}

func (t *Attr) Type() TargetType {
	return TargetAttr
}
