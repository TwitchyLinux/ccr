package buildstep

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/twitchylinux/ccr/vts"
)

func RunUnpackGz(rb RunningBuild, step *vts.BuildStep) error {
	gz, err := rb.SourceFS().Open(step.Path)
	if err != nil {
		return err
	}
	defer gz.Close()
	tape, err := gzip.NewReader(gz)
	if err != nil {
		return fmt.Errorf("reading gzip: %v", err)
	}

	fs := rb.BuildFS()
	if err := fs.MkdirAll(filepath.Join(rb.Dir(), step.ToPath), 0755); err != nil && !os.IsExist(err) {
		return fmt.Errorf("mkdir to %q: %v", step.ToPath, err)
	}

	tr := tar.NewReader(tape)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("reading tar: %v", err)
		}

		switch header.Typeflag {
		case tar.TypeDir:
			if err := fs.MkdirAll(filepath.Join(rb.Dir(), step.ToPath, header.Name), header.FileInfo().Mode()); err != nil && !os.IsExist(err) {
				return fmt.Errorf("mkdir %q: %v", header.Name, err)
			}
		case tar.TypeReg:
			outFile, err := fs.OpenFile(filepath.Join(rb.Dir(), step.ToPath, header.Name), os.O_WRONLY|os.O_CREATE|os.O_EXCL|os.O_TRUNC, header.FileInfo().Mode())
			if err != nil {
				return fmt.Errorf("open %q: %v", header.Name, err)
			}
			if _, err := io.Copy(outFile, tr); err != nil {
				return fmt.Errorf("copying %q: %v", header.Name, err)
			}
			outFile.Close()

		default:
			return fmt.Errorf("unsupported tar resource: %x", header.Typeflag)
		}
	}
	return nil
}
