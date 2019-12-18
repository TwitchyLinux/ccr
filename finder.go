package ccr

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/twitchylinux/ccr/vts"
	"github.com/twitchylinux/ccr/vts/ccbuild"
)

// A CCRResolver returns a target at the given path. If a target is not present
// under the resolver but no other error occurred, the resolver should return
// os.ErrNotExist.
type CCRResolver func(fqPath string) (vts.Target, error)

// NewDirResolver constructs a resolver that reads targets from the directory
// tree under the path provided.
func NewDirResolver(dir string) *DirResolver {
	return &DirResolver{
		dir:     dir,
		targets: make(map[string]vts.GlobalTarget, 32),
	}
}

// DirResolver resolves targets laid out as children of a directory.
type DirResolver struct {
	dir     string
	targets map[string]vts.GlobalTarget
}

// Resolve returns the target addressesd by the given target path.
func (r *DirResolver) Resolve(fqPath string) (vts.Target, error) {
	if t, ok := r.targets[fqPath]; ok {
		return t, nil
	}

	if !strings.HasPrefix(fqPath, "//") {
		return nil, fmt.Errorf("non-absolute path %q cannot be resolved by a directory resolver", fqPath)
	}
	cIdx := strings.Index(fqPath, ":")
	if cIdx < 0 {
		return nil, errors.New("no target specified")
	}

	p := fqPath[2:cIdx]
	fPath := filepath.Join(r.dir, p+".ccr")
	d, err := ioutil.ReadFile(fPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrNotExists(fqPath)
		}
		return nil, err
	}

	s, err := ccbuild.NewScript(d, fqPath[:cIdx], fPath, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("%s: %v", fqPath, err)
	}
	for _, t := range s.Targets() {
		if gt, ok := t.(vts.GlobalTarget); ok {
			r.targets[gt.GlobalPath()] = gt
		}
	}
	if t, ok := r.targets[fqPath]; ok {
		return t, nil
	}
	return nil, ErrNotExists(fqPath)
}

// FindOptions describes how targets referenced by path should be found.
type FindOptions struct {
	PrefixResolvers   map[string]CCRResolver
	FallbackResolvers []CCRResolver
}

// Find searches through configured resolvers to find the target with
// the provided path.
func (o *FindOptions) Find(path string) (vts.Target, error) {
	if path == "" {
		return nil, errors.New("cannot find target at empty path")
	}

	if !strings.HasPrefix(path, "//") { // Working tree paths can't be a prefix resolver.
		spl := strings.Split(path, "://")
		if resolve, ok := o.PrefixResolvers[spl[0]]; ok {
			t, err := resolve(path)
			if err == os.ErrNotExist {
				return nil, ErrNotExists(path)
			}
			return t, err
		}
	}

	for _, r := range o.FallbackResolvers {
		target, err := r(path)
		if err == os.ErrNotExist {
			continue
		}
		return target, err
	}

	return nil, ErrNotExists(path)
}
