package gen

import (
	"io"
	"os"
	"path/filepath"

	"github.com/twitchylinux/ccr/vts"
	"gopkg.in/src-d/go-billy.v4"
)

func fileSrcInfo(resource *vts.Resource, src *vts.Puesdo, env *vts.RunnerEnv) (string, string, os.FileMode, error) {
	outFilePath, err := determinePath(resource, env)
	if err != nil {
		return "", "", 0, err
	}
	srcFilePath := filepath.Join(filepath.Dir(src.ContractPath), src.Path)
	if src.Host {
		srcFilePath = src.Path
	}

	mode, err := determineMode(resource, env)
	switch {
	case err == errNoAttr:
		st, err := os.Stat(srcFilePath)
		if err != nil {
			return "", "", 0, vts.WrapWithPath(err, srcFilePath)
		}
		mode = st.Mode() & os.ModePerm
	case err != nil:
		return "", "", 0, err
	}
	return srcFilePath, outFilePath, mode, nil
}

// GenerateFile implements generation of a resource target, for resources
// backed by a file source.
func GenerateFile(gc GenerationContext, resource *vts.Resource, src *vts.Puesdo) error {
	srcPath, outPath, mode, err := fileSrcInfo(resource, src, gc.RunnerEnv)
	if err != nil {
		return err
	}
	return generateFile(gc.RunnerEnv.FS, srcPath, outPath, mode)
}

func generateFile(fs billy.Filesystem, srcPath, outPath string, mode os.FileMode) error {
	r, err := os.OpenFile(srcPath, os.O_RDONLY, 0644)
	if err != nil {
		return vts.WrapWithPath(err, srcPath)
	}
	defer r.Close()

	w, err := fs.OpenFile(outPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, mode)
	if err != nil {
		return vts.WrapWithPath(err, outPath)
	}
	defer w.Close()
	if _, err := io.Copy(w, r); err != nil {
		return vts.WrapWithPath(err, outPath)
	}
	return nil
}
