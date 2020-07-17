package runners

import (
	"crypto/sha256"
	"fmt"
	"net/url"

	version "github.com/knqyf263/go-deb-version"
	"github.com/twitchylinux/ccr/vts"
	"go.starlark.net/starlark"
)

// DebInfoCheckValid returns a runner that can check attrs
// are valid debian package info structures.
func DebInfoCheckValid() *debInfoValidRunner {
	return &debInfoValidRunner{}
}

type debInfoValidRunner struct{}

func (*debInfoValidRunner) Kind() vts.CheckerKind { return vts.ChkKindEachAttr }

func (*debInfoValidRunner) String() string { return "attr.deb_info_valid" }

func (*debInfoValidRunner) Freeze() {}

func (*debInfoValidRunner) Truth() starlark.Bool { return true }

func (*debInfoValidRunner) Type() string { return "runner" }

func (t *debInfoValidRunner) Hash() (uint32, error) {
	h := sha256.Sum256([]byte(fmt.Sprintf("%p", t)))
	return uint32(uint32(h[0]) + uint32(h[1])<<8 + uint32(h[2])<<16 + uint32(h[3])<<24), nil
}

func (*debInfoValidRunner) Run(attr *vts.Attr, chkr *vts.Checker, opts *vts.RunnerEnv) error {
	v, err := attr.Value(opts)
	if err != nil {
		return err
	}
	info, ok := v.(*starlark.Dict)
	if !ok {
		return fmt.Errorf("expected list, got %T", v)
	}

	for _, k := range info.Keys() {
		kStr, ok := k.(starlark.String)
		if !ok {
			return fmt.Errorf("key %q is of type %T", k.String(), k)
		}
		v, _, _ := info.Get(k)

		switch string(kStr) {
		case "name", "maintainer", "section", "description", "priority":
			if _, ok := v.(starlark.String); !ok {
				return fmt.Errorf("%s is not a string, got type %T", string(kStr), v)
			}

		case "homepage":
			if vStr, ok := v.(starlark.String); ok {
				url, err := url.Parse(string(vStr))
				if err != nil {
					return fmt.Errorf("invalid homepage URL: %v", err)
				}
				if url.Scheme != "https" && url.Scheme != "http" {
					return fmt.Errorf("homepage URL has invalid scheme: %v", url.Scheme)
				}
			} else {
				return fmt.Errorf("homepage is not a string, got type %T", v)
			}

		case "version":
			if vStr, ok := v.(starlark.String); ok {
				if _, err := version.NewVersion(string(vStr)); err != nil {
					return fmt.Errorf("invalid version string: %v", err)
				}
			} else {
				return fmt.Errorf("version is not a string, got type %T", v)
			}

		case "pre-depends-on", "depends-on", "breaks", "replaces":
			set, ok := v.(*starlark.List)
			if !ok {
				return fmt.Errorf("expected list for key %s, got %T", string(kStr), v)
			}
			if err := checkDepsList(set, string(kStr)); err != nil {
				return err
			}
		default:
			return fmt.Errorf("unexpected key: %s", string(kStr))
		}
	}
	return nil
}

func checkDepsList(set *starlark.List, keyName string) error {
	i := set.Iterate()
	defer i.Done()
	var x starlark.Value
	for i.Next(&x) {
		d, ok := x.(*starlark.Dict)
		if !ok {
			return fmt.Errorf("%s entry is not a dictionary", keyName)
		}
		for _, k2 := range d.Keys() {
			kStr2, ok := k2.(starlark.String)
			if !ok {
				return fmt.Errorf("%s entry has sub-key %q is of type %T", keyName, string(kStr2), k2)
			}
			v, _, _ := d.Get(k2)
			switch string(kStr2) {
			case "name":
				if _, ok := v.(starlark.String); !ok {
					return fmt.Errorf("%s entry has unexpected sub-key %q value type %T", keyName, string(kStr2), v)
				}
			case "version":
				s2, ok := v.(starlark.String)
				if !ok {
					return fmt.Errorf("%s entry has unexpected sub-key %q value type %T", keyName, string(kStr2), v)
				}
				if _, err := version.NewVersion(string(s2)); err != nil {
					return fmt.Errorf("%s entry has invalid version: %v", keyName, err)
				}
			case "version-constraint":
				s2, ok := v.(starlark.String)
				if !ok {
					return fmt.Errorf("%s entry has unexpected sub-key %q value type %T", keyName, string(kStr2), v)
				}
				switch string(s2) {
				case "=", ">=", "<=", ">>", "<<":
				default:
					return fmt.Errorf("%s entry has unexpected version-constraint value %s", keyName, string(s2))
				}
			default:
				return fmt.Errorf("%s entry has unexpected sub-key %q", keyName, string(kStr2))
			}
		}
	}

	return nil
}
