package syslib

import (
	"github.com/twitchylinux/ccr/vts"
)

const (
	// Key against which a map of all system libs are stored.
	libDirsKey = "syslib-dirs"
	// Key against which a map of all binaries are stored.
	binariesKey = "syslib-binaries"
)

func getLibraryDirs(opts *vts.RunnerEnv) (map[string]*vts.Resource, error) {
	return getPathResourceCollection(libDirsKey, "common://resources:library_dir", opts)
}

func getBinaries(opts *vts.RunnerEnv) (map[string]*vts.Resource, error) {
	return getPathResourceCollection(binariesKey, "common://resources:binary", opts)
}

func getPathResourceCollection(key, classPath string, opts *vts.RunnerEnv) (map[string]*vts.Resource, error) {
	if d, ok := opts.Universe.GetData(key); ok {
		return d.(map[string]*vts.Resource), nil
	}

	// Information is not cached, lets compute it.
	pathResources := make(map[string]*vts.Resource, 64)
	for _, target := range opts.Universe.AllTargets() {
		if r, isResource := target.(*vts.Resource); isResource {
			if parent := r.Parent.Target.(*vts.ResourceClass); parent.GlobalPath() == classPath {
				path, err := resourcePath(r, opts)
				if err != nil {
					return nil, vts.WrapWithTarget(err, r)
				}
				pathResources[path] = r
			}
		}
	}
	opts.Universe.SetData(key, pathResources)

	return pathResources, nil
}
