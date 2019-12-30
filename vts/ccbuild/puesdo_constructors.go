package ccbuild

import (
	"github.com/twitchylinux/ccr/vts"
	"go.starlark.net/starlark"
)

func makePuesdotarget(s *Script, kind vts.PuesdoKind) *starlark.Builtin {
	t := vts.TargetPuesdo

	return starlark.NewBuiltin(t.String(), func(thread *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		var path, sha256, url string
		if err := starlark.UnpackArgs(t.String(), args, kwargs, "path?", &path, "sha256?", &sha256, "url?", &url); err != nil {
			return starlark.None, err
		}

		pt := &vts.Puesdo{
			Kind:         kind,
			Path:         path,
			SHA256:       sha256,
			URL:          url,
			ContractPath: s.fPath,
			Pos: &vts.DefPosition{
				Path:  s.fPath,
				Frame: thread.CallFrame(1),
			},
		}

		return pt, nil
	})
}
