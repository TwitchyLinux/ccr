package info

import (
	"fmt"
	"os"

	"github.com/twitchylinux/ccr/vts"
)

// Stat populator data keys.
const (
	FileStat = "stat"
	FilePath = "path"
)

// FileInfo describes information about a path.
type FileInfo struct {
	os.FileInfo
	Path string
}

type statPopulator struct{}

func (i *statPopulator) Name() string {
	return "Stat Populator"
}

func (i *statPopulator) Run(t vts.Target, opts *vts.RunnerEnv, info *vts.RuntimeInfo) error {
	r, ok := t.(*vts.Resource)
	if !ok {
		return fmt.Errorf("info.statPopulator can only operate on resource targets, got %T", t)
	}
	path, err := resourcePath(r, opts)
	if err != nil {
		return err
	}
	info.Set(i, FilePath, path)
	stat, err := opts.FS.Stat(path)
	if err != nil {
		return vts.WrapWithPath(err, path)
	}

	info.Set(i, FileStat, FileInfo{
		FileInfo: stat,
		Path:     path,
	})
	return nil
}
