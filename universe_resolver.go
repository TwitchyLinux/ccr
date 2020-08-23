package ccr

import (
	"os"
	"path/filepath"

	"github.com/twitchylinux/ccr/vts"
	"github.com/twitchylinux/ccr/vts/common"
	"go.starlark.net/starlark"
)

// runtimeResolver implements vts.UniverseResolver.
type runtimeResolver struct {
	*Universe
	globalData map[string]interface{}
}

func symlinkTarget(t vts.Target, env *vts.RunnerEnv) (string, bool) {
	rt, ok := t.(*vts.Resource)
	if !ok {
		return "", false
	}
	if rt.Parent.Target != common.SymlinkResourceClass {
		return "", false
	}
	v, err := determineAttrValue(rt, common.TargetClass, env)
	if err != nil {
		return "", false
	}
	if s, ok := v.(starlark.String); ok {
		return filepath.Clean(string(s)), true
	}
	return "", false
}

// Inject wires a new, runtime-generated target into the universe.
func (u *runtimeResolver) Inject(t vts.Target) (vts.Target, error) {
	return t, u.linkTarget(t)
}

// FindByPath retunrs the unit which declares the given path attribute.
func (u *runtimeResolver) FindByPath(path string, env *vts.RunnerEnv) (vts.Target, error) {
	t, ok := u.pathTargets[path]
	if !ok {
		// Maybe theres a symlink we need to check out.
		dir, base := filepath.Dir(path), filepath.Base(path)
		if dt, ok := u.pathTargets[dir]; ok {
			if target, ok := symlinkTarget(dt, env); ok {
				return u.FindByPath(filepath.Join(dir, "../", target, base), env)
			}
		}

		return nil, os.ErrNotExist
	}
	return t, nil
}

// AllTargets returns all targets which were enumerated when building the
// universe.
func (u *runtimeResolver) AllTargets() []vts.GlobalTarget {
	return u.allTargets
}

// GetData returns the arbitrary value previously stored against key, or
// false if no such key exists.
func (u *runtimeResolver) GetData(key string) (interface{}, bool) {
	v, ok := u.globalData[key]
	return v, ok
}

// SetData stores arbitrary data against the given key, overwriting any
// existing value. This data can be later retrieved with GetData().
func (u *runtimeResolver) SetData(key string, data interface{}) {
	u.globalData[key] = data
}
