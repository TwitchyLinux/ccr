package proc

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"fmt"
	"io/ioutil"
	"strings"

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

func scriptFromInline(d []byte) ([]byte, error) {
	var s bytes.Buffer
	s.Grow(128)
	s.WriteString("def inline_fn(attr, t):\n")

	spl, had1stLine, indent := strings.Split(strings.TrimRight(string(d), "\n "), "\n"), false, 0
	for i, line := range spl {
		if line == "" {
			continue
		}
		// Count indents on the first line, trim from subsequent.
		if !had1stLine {
			had1stLine = true
			for i := 0; i < len(line) && line[i] == ' '; i++ {
				indent++
			}
		}
		s.WriteString("  ")

		if len(line) < indent || line[:indent] != strings.Repeat(" ", indent) {
			return nil, errors.New("inline script had inconsistent indentation")
		}
		if i+1 == len(spl) && !strings.HasPrefix(line[indent:], "return ") {
			s.WriteString("return ")
		}
		s.WriteString(line[indent:] + "\n")
	}
	return s.Bytes(), nil
}

// EvalComputedAttribute computes the value of an attribute whose value is wired to a function.
func EvalComputedAttribute(attr *vts.Attr, target vts.Target, runInfo *vts.ComputedValue, env *vts.RunnerEnv) (v starlark.Value, err error) {
	var (
		out             = &proc{dir: runInfo.ContractDir, fPath: runInfo.Filename, readOnly: !runInfo.ReadWrite}
		funcName string = runInfo.Func
		d        []byte
	)
	if len(runInfo.InlineScript) > 0 {
		if d, err = scriptFromInline(runInfo.InlineScript); err != nil {
			return starlark.None, err
		}
		funcName = "inline_fn"
	} else {
		if d, err = ioutil.ReadFile(runInfo.Filename); err != nil {
			return starlark.None, err
		}
	}

	defer func() {
		if out.env != nil {
			if err2 := out.env.Close(); err2 != nil && err != nil {
				err = fmt.Errorf("closing env: %v", err2)
			}
		}
	}()

	out.thread, out.globals, err = out.loadScript(d, runInfo.Filename, out)
	if err != nil {
		return starlark.None, err
	}
	defer out.Close()

	fn, exists := out.globals[funcName]
	if !exists {
		return starlark.None, fmt.Errorf("cannot compute value: function %q was not present", runInfo.Func)
	}

	return starlark.Call(out.thread, fn, starlark.Tuple{&computedAttrProxy{attr: attr}, proxyTarget(target)}, nil)
}
