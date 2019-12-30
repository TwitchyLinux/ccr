package ccbuild

import (
	"fmt"

	"github.com/twitchylinux/ccr/vts"
	"go.starlark.net/starlark"
)

func makePuesdotarget(s *Script, kind vts.PuesdoKind) *starlark.Builtin {
	t := vts.TargetPuesdo

	return starlark.NewBuiltin(t.String(), func(thread *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		var path, sha256, url string
		var name string
		var details *starlark.List
		if err := starlark.UnpackArgs(t.String(), args, kwargs, "path?", &path,
			"name?", &name, "details?", &details,
			"sha256?", &sha256, "url?", &url); err != nil {
			return starlark.None, err
		}

		pt := &vts.Puesdo{
			Kind:         kind,
			TargetPath:   s.makePath(name),
			Name:         name,
			ContractPath: s.fPath,
			Pos: &vts.DefPosition{
				Path:  s.fPath,
				Frame: thread.CallFrame(1),
			},

			Path:   path,
			SHA256: sha256,
			URL:    url,
		}

		if details != nil {
			i := details.Iterate()
			defer i.Done()
			var x starlark.Value
			for i.Next(&x) {
				v, err := toDetailsTarget(s.path, x)
				if err != nil {
					return nil, fmt.Errorf("invalid detail: %v", err)
				}
				pt.Details = append(pt.Details, v)
			}
		}

		// If theres no name, it must be an anonymous target as part of another
		// target. We don't add it to the targets list.
		if name == "" {
			return pt, nil
		}
		s.targets = append(s.targets, pt)
		return starlark.None, nil
	})
}
