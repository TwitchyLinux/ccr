// Package ccbuild interprets .ccr files.
package ccbuild

import (
	"errors"
	"fmt"
	"strings"

	"github.com/twitchylinux/ccr/vts"
	"go.starlark.net/starlark"
)

// ScriptLoader provides a means for arbitrary imports to be resolved.
type ScriptLoader interface {
	resolveImport(name string) ([]byte, error)
}

// Script represents a .ccr file execution.
type Script struct {
	thread  *starlark.Thread
	globals starlark.StringDict

	path    string
	fPath   string
	targets []vts.Target
}

// Close shuts down all resources associated with the script.
func (s *Script) Close() error {
	return nil
}

// Targets returns a list of all globally-reachable targets.
func (s *Script) Targets() []vts.Target {
	return s.targets
}

func (s *Script) loadScript(script []byte, fname string, loader ScriptLoader) (*starlark.Thread, starlark.StringDict, error) {
	var moduleCache = map[string]starlark.StringDict{}
	var load func(_ *starlark.Thread, module string) (starlark.StringDict, error)

	builtins, err := s.makeBuiltins()
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

// NewScript initializes a new .ccr interpreter. The data parameter should
// contain the contents of the ccr file, and the targetPath parameter should
// represent the CCR path to the file.
func NewScript(data []byte, targetPath, fPath string, loader ScriptLoader, printer func(string)) (*Script, error) {
	return makeScript(data, targetPath, fPath, loader, nil, printer)
}

func makeScript(data []byte, targetPath, fPath string, loader ScriptLoader,
	testHook func(thread *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error),
	printer func(string)) (*Script, error) {
	out := &Script{
		path:  targetPath,
		fPath: fPath,
	}

	var err error
	out.thread, out.globals, err = out.loadScript(data, targetPath, out)
	if err != nil {
		return nil, err
	}

	return out, nil
}

func (s *Script) resolveImport(path string) ([]byte, error) {
	return nil, errors.New("not implemented")
}

func (s *Script) makePath(targetName string) string {
	if targetName == "" {
		return ""
	}
	if strings.HasPrefix(targetName, ":") {
		return s.path + targetName
	}
	return fmt.Sprintf("%s:%s", s.path, targetName)
}
