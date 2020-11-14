// Package proc manages execution of procedures.
package proc

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"strings"

	"github.com/twitchylinux/ccr/vts"
	"go.starlark.net/starlark"
	"go.starlark.net/starlarkstruct"
)

type scriptLoader interface {
	resolveImport(name string) ([]byte, error)
}

// proc wraps Env to facilitate execution of a procedure or macro.
type proc struct {
	thread  *starlark.Thread
	globals starlark.StringDict

	readOnly bool
	env      *Env

	dir   string
	fPath string
}

// Close shuts down all resources associated with the script.
func (p *proc) Close() error {
	if p.env != nil {
		return p.env.Close()
	}
	return nil
}

func (p *proc) loadScript(script []byte, fname string, loader scriptLoader) (*starlark.Thread, starlark.StringDict, error) {
	var moduleCache = map[string]starlark.StringDict{}
	var load func(_ *starlark.Thread, module string) (starlark.StringDict, error)

	builtins, err := p.makeBuiltins()
	if err != nil {
		return nil, nil, err
	}

	load = func(_ *starlark.Thread, module string) (starlark.StringDict, error) {
		m, ok := moduleCache[module]
		if m == nil && ok {
			return nil, errors.New("cycle in dependency graph when loading " + module)
		}
		if m != nil {
			return m, nil
		}

		// loading in progress
		moduleCache[module] = nil
		d, err2 := loader.resolveImport(module)
		if err2 != nil {
			return nil, err2
		}
		thread := &starlark.Thread{
			Load: load,
		}

		mod, err2 := starlark.ExecFile(thread, module, d, builtins)
		if err2 != nil {
			return nil, err2
		}
		moduleCache[module] = mod
		return mod, nil
	}

	thread := &starlark.Thread{
		Load: load,
	}

	globals, err := starlark.ExecFile(thread, fname, script, builtins)
	if err != nil {
		return nil, nil, err
	}

	return thread, globals, nil
}

func (p *proc) runBuiltin() *starlark.Builtin {
	return starlark.NewBuiltin("run", func(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, _ []starlark.Tuple) (starlark.Value, error) {
		var err error
		if p.env == nil {
			if p.env, err = NewEnv(p.readOnly); err != nil {
				return starlark.None, fmt.Errorf("initializing environment: %v", err)
			}
		}

		vals := make([]string, len(args))
		for i, a := range args {
			vals[i], _ = starlark.AsString(a)
		}

		out, sErr, code, err := p.env.RunBlocking(p.dir, nil, vals...)
		if err != nil && code == 0 {
			return starlark.None, err
		}

		return starlarkstruct.FromStringDict(starlarkstruct.Default, starlark.StringDict{
			"output":    starlark.String(out),
			"stderr":    starlark.String(sErr),
			"exit_code": starlark.MakeInt(code),
		}), nil
	})
}

func (p *proc) makeBuiltins() (starlark.StringDict, error) {
	return starlark.StringDict{
		"none": starlark.None,
		"run":  p.runBuiltin(),
		"broken_assumption": starlark.NewBuiltin("broken_assumption", func(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, _ []starlark.Tuple) (starlark.Value, error) {
			vals := make([]string, len(args))
			for i, a := range args {
				vals[i], _ = starlark.AsString(a)
			}
			return starlark.None, fmt.Errorf("broken assumption: %s", strings.Join(vals, " "))
		}),
	}, nil
}

func (p *proc) resolveImport(path string) ([]byte, error) {
	return nil, errors.New("not implemented")
}

type target interface {
	vts.Target
}

// targetProxy proxies access to a vts.Target.
type targetProxy struct {
	t         target
	hasNaming bool
	hasDeps   bool
	hasAttrs  bool
	hasParent bool
}

func proxyTarget(t vts.Target) *targetProxy {
	out := targetProxy{t: t.(target)}
	_, out.hasNaming = t.(vts.GlobalTarget)
	_, out.hasDeps = t.(vts.DepTarget)
	_, out.hasAttrs = t.(vts.DetailedTarget)
	_, out.hasParent = t.(vts.ClassedTarget)
	return &out
}

func (p *targetProxy) String() string {
	return p.t.TargetType().String()
}

// Type implements starlark.Value.
func (p *targetProxy) Type() string {
	return p.String()
}

// Freeze implements starlark.Value.
func (p *targetProxy) Freeze() {
}

// Truth implements starlark.Value.
func (p *targetProxy) Truth() starlark.Bool {
	return p.t != nil
}

// Hash implements starlark.Value.
func (p *targetProxy) Hash() (uint32, error) {
	h := sha256.Sum256([]byte(p.String()))
	return uint32(uint32(h[0]) + uint32(h[1])<<8 + uint32(h[2])<<16 + uint32(h[3])<<24), nil
}

// AttrNames implements starlark.Value.
func (p *targetProxy) AttrNames() []string {
	out := make([]string, 2, 8)
	out[0] = "type"
	out[1] = "is_class"
	if p.hasNaming {
		out = append(out, "name", "path")
	}
	if p.hasDeps {
		out = append(out, "deps")
	}
	if p.hasAttrs {
		out = append(out, "details")
	}
	return out
}

// Attr implements starlark.Value.
func (p *targetProxy) Attr(name string) (starlark.Value, error) {
	if p.t != nil {
		switch name {
		case "type":
			return starlark.String(p.t.TargetType().String()), nil
		case "is_class":
			return starlark.Bool(p.t.IsClassTarget()), nil
		}

		switch {
		case p.hasNaming && name == "name":
			return starlark.String(p.t.(vts.GlobalTarget).TargetName()), nil
		case p.hasNaming && name == "path":
			return starlark.String(p.t.(vts.GlobalTarget).GlobalPath()), nil
		case p.hasParent && name == "parent":
			return proxyTarget(p.t.(vts.ClassedTarget).Class().Target), nil
		case p.hasDeps && name == "deps":
			deps := p.t.(vts.DepTarget).Dependencies()
			out := make([]starlark.Value, len(deps))
			for i, d := range deps {
				out[i] = proxyTarget(d.Target)
			}
			return starlark.NewList(out), nil
		case p.hasAttrs && name == "details":
			deets := p.t.(vts.DetailedTarget).Attributes()
			out := make([]starlark.Value, len(deets))
			for i, d := range deets {
				out[i] = proxyTarget(d.Target)
			}
			return starlark.NewList(out), nil
		}
	}

	return nil, starlark.NoSuchAttrError(
		fmt.Sprintf("%s has no .%s attribute", p.Type(), name))
}
