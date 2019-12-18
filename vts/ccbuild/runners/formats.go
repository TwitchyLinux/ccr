// Package runners implements builtin runners.
package runners

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/twitchylinux/ccr/vts"
	"go.starlark.net/starlark"
)

// JSONCheckValid returns a runner that can check resources
// are correctly JSON formatted.
func JSONCheckValid() *jsonValidRunner {
	return &jsonValidRunner{}
}

type jsonValidRunner struct{}

func (*jsonValidRunner) Kind() vts.CheckerKind { return vts.ChkKindEachResource }

func (*jsonValidRunner) String() string { return "json.check_valid" }

func (*jsonValidRunner) Freeze() {}

func (*jsonValidRunner) Truth() starlark.Bool { return true }

func (*jsonValidRunner) Type() string { return "runner" }

func (t *jsonValidRunner) Hash() (uint32, error) {
	h := sha256.Sum256([]byte(fmt.Sprintf("%p", t)))
	return uint32(uint32(h[0]) + uint32(h[1])<<8 + uint32(h[2])<<16 + uint32(h[3])<<24), nil
}

func (*jsonValidRunner) Run(resource *vts.Resource, opts *vts.CheckerOpts) error {
	path, err := resourcePath(resource)
	if err != nil {
		return err
	}
	f, err := os.Open(filepath.Join(opts.Dir, path))
	if err != nil {
		return err
	}
	defer f.Close()

	var o map[string]interface{}
	return json.NewDecoder(f).Decode(&o)
}
