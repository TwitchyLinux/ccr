package ccr

import (
	"os"

	"github.com/twitchylinux/ccr/vts"
)

// runtimeResolver implements vts.UniverseResolver.
type runtimeResolver struct {
	*Universe
	globalData map[string]interface{}
}

// FindByPath retunrs the unit which declares the given path attribute.
func (u *runtimeResolver) FindByPath(path string) (vts.Target, error) {
	t, ok := u.pathTargets[path]
	if !ok {
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
