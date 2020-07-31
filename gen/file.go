package gen

import (
	"archive/tar"
	"errors"
	"os"
	"path/filepath"

	"github.com/twitchylinux/ccr/vts"
)

func filesetForFileSource(src *vts.Puesdo) (*unaryFileset, error) {
	p := filepath.Join(filepath.Dir(src.ContractPath), src.Path)
	s, err := os.Stat(p)
	if err != nil {
		return nil, err
	}
	return &unaryFileset{
		srcPath: p,
		srcHeader: tar.Header{
			AccessTime: s.ModTime(),
			ChangeTime: s.ModTime(),
			Mode:       int64(s.Mode()),
			Size:       s.Size(),
			Typeflag:   tar.TypeReg,
		},
	}, nil
}

// resourcePathMode returns the path and mode set on a resource via attributes
// of the path and mode attribute classes. errNoAttr is returned if no path
// is specified, but no mode being specified returns no error and instead a
// mode value of 0.
func resourcePathMode(resource *vts.Resource, env *vts.RunnerEnv) (string, os.FileMode, error) {
	outFilePath, err := determinePath(resource, env)
	if err != nil {
		return "", 0, err
	}

	mode, err := determineMode(resource, env)
	switch {
	case errors.Is(err, errNoAttr):
		mode = 0
	case err != nil:
		return "", 0, err
	}
	return outFilePath, mode, nil
}
