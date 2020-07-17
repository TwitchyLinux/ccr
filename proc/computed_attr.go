package proc

import (
	"crypto/sha256"
	"fmt"
	"io/ioutil"

	"github.com/twitchylinux/ccr/vts"
	"go.starlark.net/starlark"
)

// computedAttrProxy proxies access to a *vts.Attr, where
// Val.(type) == vts.ComputedValue.
type computedAttrProxy struct {
	attr *vts.Attr
}

func (p *computedAttrProxy) String() string {
	return p.attr.String()
}

// Type implements starlark.Value.
func (p *computedAttrProxy) Type() string {
	return p.attr.Type()
}

// Freeze implements starlark.Value.
func (p *computedAttrProxy) Freeze() {
}

// Truth implements starlark.Value.
func (p *computedAttrProxy) Truth() starlark.Bool {
	return p.attr != nil
}

// Hash implements starlark.Value.
func (p *computedAttrProxy) Hash() (uint32, error) {
	h := sha256.Sum256([]byte(p.String()))
	return uint32(uint32(h[0]) + uint32(h[1])<<8 + uint32(h[2])<<16 + uint32(h[3])<<24), nil
}

// AttrNames implements starlark.Value.
func (p *computedAttrProxy) AttrNames() []string {
	return []string{"name", "path", "parent"}
}

// Attr implements starlark.Value.
func (p *computedAttrProxy) Attr(name string) (starlark.Value, error) {
	switch name {
	// case "mkdir":
	// 	return starlark.NewBuiltin("mkdir", func(thread *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	// 		var path starlark.String
	// 		if err := starlark.UnpackArgs("mkdir", args, kwargs, "path", &path); err != nil {
	// 			return starlark.None, err
	// 		}
	// 		return starlark.None, p.fs.Mkdir(string(path))
	// 	}), nil

	case "parent":
		return proxyTarget(p.attr.Parent.Target), nil
	case "path":
		return starlark.String(p.attr.Path), nil
	case "name":
		return starlark.String(p.attr.Name), nil
	}

	return nil, starlark.NoSuchAttrError(
		fmt.Sprintf("%s has no .%s attribute", p.Type(), name))
}

// EvalComputedAttribute computes the value of an attribute whose value is wired to a function.
func EvalComputedAttribute(attr *vts.Attr, target vts.Target, runInfo *vts.ComputedValue, env *vts.RunnerEnv) (starlark.Value, error) {
	var (
		err error
		out = &proc{fPath: runInfo.Filename}
		d   []byte
	)
	if d, err = ioutil.ReadFile(runInfo.Filename); err != nil {
		return starlark.None, err
	}

	out.thread, out.globals, err = out.loadScript(d, runInfo.Filename, out)
	if err != nil {
		return starlark.None, err
	}
	defer out.Close()

	fn, exists := out.globals[runInfo.Func]
	if !exists {
		return starlark.None, fmt.Errorf("cannot compute value: function %q was not present", runInfo.Func)
	}

	return starlark.Call(out.thread, fn, starlark.Tuple{&computedAttrProxy{attr: attr}, proxyTarget(target)}, nil)
}
