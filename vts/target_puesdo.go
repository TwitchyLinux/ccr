package vts

import (
	"crypto/sha256"
	"errors"
	"fmt"

	"go.starlark.net/starlark"
)

type PuesdoKind string

// Valid types of puesdo targets.
const (
	FileRef PuesdoKind = "file"
	DebRef  PuesdoKind = "deb"
)

// Puesdo is a special case target.
type Puesdo struct {
	Kind         PuesdoKind
	Pos          *DefPosition
	Name         string
	TargetPath   string
	ContractPath string

	// Applicable to FileRef & DebRef targets.
	Path string

	// Applicable to DebRef targets.
	SHA256 string
	URL    string

	// Applicable to FileRef targets.
	Host bool

	Details []TargetRef
}

func (t *Puesdo) DefinedAt() *DefPosition {
	return t.Pos
}

func (t *Puesdo) IsClassTarget() bool {
	return false
}

func (t *Puesdo) GlobalPath() string {
	return t.TargetPath
}

func (t *Puesdo) TargetName() string {
	return t.Name
}

func (t *Puesdo) TargetType() TargetType {
	return TargetPuesdo
}

func (t *Puesdo) Attributes() []TargetRef {
	return t.Details
}

func (t *Puesdo) Validate() error {
	if err := validateDetails(t.Details); err != nil {
		return err
	}

	switch t.Kind {
	case DebRef:
		if t.URL == "" && t.Path == "" {
			return errors.New("deb source must specify a URL or path")
		}
		if t.SHA256 == "" {
			return errors.New("deb source must specify a sha256 hash")
		}
		if t.Host {
			return errors.New("host can only be set on file sources")
		}
		return nil
	case FileRef:
		if t.Path == "" {
			return errors.New("file source cannot have empty path")
		}
		return nil
	default:
		return fmt.Errorf("unrecognized puesdotarget: %v", t.Kind)
	}
}

func (t *Puesdo) String() string {
	return fmt.Sprintf("puesdo<%s>", t.Kind)
}

func (t *Puesdo) Freeze() {

}

func (t *Puesdo) Truth() starlark.Bool {
	return true
}

func (t *Puesdo) Type() string {
	return "puesdo"
}

func (t *Puesdo) Hash() (uint32, error) {
	h := sha256.Sum256([]byte(fmt.Sprintf("%p", t)))
	return uint32(uint32(h[0]) + uint32(h[1])<<8 + uint32(h[2])<<16 + uint32(h[3])<<24), nil
}

func (t *Puesdo) RollupHash(env *RunnerEnv, eval computeEval) ([]byte, error) {
	hash := sha256.New()
	fmt.Fprintf(hash, "%q\n%q\n%q\n", t.Kind, t.Name, t.TargetPath)
	fmt.Fprintf(hash, "%q\n%q\n%q\n", t.Path, t.URL, t.SHA256)
	fmt.Fprintf(hash, "%v\n", t.Host)

	for _, attr := range t.Details {
		a := attr.Target.(*Attr)
		fmt.Fprintf(hash, "%q\n%q\n%q\n", a.Name, a.Path, a.Parent.Target.(*AttrClass).GlobalPath())
		// TODO: Hash attribute class.
		if cv, isComputedValue := a.Val.(*ComputedValue); isComputedValue {
			fmt.Fprintf(hash, "computed params: file = %q func = %q inline = %q", cv.Filename, cv.Func, string(cv.InlineScript))
		}
		v, err := a.Value(t, env, eval)
		if err != nil {
			return nil, WrapWithTarget(err, a)
		}
		fmt.Fprint(hash, v)
	}

	return hash.Sum(nil), nil
}
