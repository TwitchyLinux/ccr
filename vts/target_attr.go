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

func (t *Attr) GlobalPath() string {
	return t.Path
}

func (t *Attr) TargetName() string {
	return t.Name
}
