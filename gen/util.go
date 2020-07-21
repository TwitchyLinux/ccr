package gen

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/twitchylinux/ccr/proc"
	"github.com/twitchylinux/ccr/vts"
	"github.com/twitchylinux/ccr/vts/common"
	"go.starlark.net/starlark"
)

var errNoAttr = errors.New("attr not specified")

func determineAttrValue(t vts.Target, cls *vts.AttrClass, env *vts.RunnerEnv) (starlark.Value, error) {
	dt, ok := t.(vts.DetailedTarget)
	if !ok {
		return nil, vts.WrapWithTarget(fmt.Errorf("no details available on target %T", t), t)
	}
	for _, attr := range dt.Attributes() {
		if attr.Target == nil {
			return nil, vts.WrapWithTarget(fmt.Errorf("unresolved target reference: %q", attr.Path), t)
		}
		a := attr.Target.(*vts.Attr)
		if a.Parent.Target == nil {
			return nil, vts.WrapWithTarget(fmt.Errorf("unresolved target reference: %q", a.Parent.Path), t)
		}
		if class := a.Parent.Target.(*vts.AttrClass); class.GlobalPath() == cls.Path {
			return a.Value(t, env, proc.EvalComputedAttribute)
		}
	}

	return nil, errNoAttr
}

func determinePath(t vts.Target, env *vts.RunnerEnv) (string, error) {
	v, err := determineAttrValue(t, common.PathClass, env)
	if err != nil {
		return "", err
	}
	if s, ok := v.(starlark.String); ok {
		return filepath.Clean(string(s)), nil
	}
	return "", vts.WrapWithTarget(fmt.Errorf("bad type for path: want string, got %T", v), t)
}

func determineMode(t vts.Target, env *vts.RunnerEnv) (os.FileMode, error) {
	v, err := determineAttrValue(t, common.ModeClass, env)
	if err != nil {
		return 0, err
	}
	if s, ok := v.(starlark.String); ok {
		mode, err := strconv.ParseInt(string(s), 8, 64)
		return os.FileMode(mode), err
	}
	return 0, vts.WrapWithTarget(fmt.Errorf("bad type for mode: want string, got %T", v), t)
}
