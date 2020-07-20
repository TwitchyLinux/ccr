package vts

// Toolchain is a target representing a specific host toolchain.
type Toolchain struct {
	Path string
	Name string
	Pos  *DefPosition

	Details        []TargetRef
	BinaryMappings map[string]string
	Deps           []TargetRef
}

func (t *Toolchain) DefinedAt() *DefPosition {
	return t.Pos
}

func (t *Toolchain) IsClassTarget() bool {
	return false
}

func (t *Toolchain) TargetType() TargetType {
	return TargetToolchain
}

func (t *Toolchain) GlobalPath() string {
	return t.Path
}

func (t *Toolchain) TargetName() string {
	return t.Name
}

func (t *Toolchain) Validate() error {
	if err := validateDetails(t.Details); err != nil {
		return err
	}
	if err := validateDeps(t.Deps); err != nil {
		return err
	}
	return nil
}

func (t *Toolchain) Dependencies() []TargetRef {
	return t.Deps
}
func (t *Toolchain) Attributes() []TargetRef {
	return t.Details
}
