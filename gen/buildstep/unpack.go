package buildstep

import (
	"archive/tar"
	"compress/bzip2"
	"compress/gzip"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/twitchylinux/ccr/cache"
	"github.com/twitchylinux/ccr/vts"
	"github.com/ulikunitz/xz"
)

// RunUnpack unpacks a .tar.gz or .tar.xz file referenced in the build step,
// into the specified directory.
func RunUnpack(c *cache.Cache, rb RunningBuild, step *vts.BuildStep) error {
	var (
		compressedStream io.ReadCloser
		err              error
	)
	switch {
	case step.Path != "":
		if compressedStream, err = rb.SourceFS().Open(step.Path); err != nil {
			return err
		}
		defer compressedStream.Close()

	case step.URL != "" && step.SHA256 != "":
		h, err := hex.DecodeString(step.SHA256)
		if err != nil {
			return err
		}
		if compressedStream, err = download(c, h, step.URL); err != nil {
			return err
		}
		defer compressedStream.Close()

	default:
		return fmt.Errorf("cannot handle non-path and non-url unpack_gz step invariant (%v)", step)
	}

	if step.Kind == vts.StepUnpackXz {
		return unpackXzReader(compressedStream, rb, step)
	}
	if step.Kind == vts.StepUnpackBz2 {
		return unpackBz2Reader(compressedStream, rb, step)
	}
	return unpackGzReader(compressedStream, rb, step)
}

func unpackGzReader(gz io.Reader, rb RunningBuild, step *vts.BuildStep) error {
	tape, err := gzip.NewReader(gz)
	if err != nil {
		return fmt.Errorf("reading gzip: %v", err)
	}
	return unpackTarReader(tape, rb, step)
}

func unpackXzReader(gz io.Reader, rb RunningBuild, step *vts.BuildStep) error {
	tape, err := xz.NewReader(gz)
	if err != nil {
		return fmt.Errorf("reading xz: %v", err)
	}
	return unpackTarReader(tape, rb, step)
}

func unpackBz2Reader(gz io.Reader, rb RunningBuild, step *vts.BuildStep) error {
	return unpackTarReader(bzip2.NewReader(gz), rb, step)
}

func unpackTarReader(tape io.Reader, rb RunningBuild, step *vts.BuildStep) error {
	fs := rb.RootFS()
	if err := fs.MkdirAll(filepath.Join(rb.OverlayUpperPath(), step.ToPath), 0755); err != nil && !os.IsExist(err) {
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
			if err := fs.MkdirAll(filepath.Join(rb.OverlayUpperPath(), step.ToPath, header.Name), header.FileInfo().Mode()); err != nil && !os.IsExist(err) {
				return fmt.Errorf("mkdir %q: %v", header.Name, err)
			}
		case tar.TypeReg:
			fp := filepath.Join(rb.OverlayUpperPath(), step.ToPath, header.Name)
			outFile, err := fs.OpenFile(fp, os.O_WRONLY|os.O_CREATE|os.O_EXCL|os.O_TRUNC, header.FileInfo().Mode())
			if err != nil {
				return fmt.Errorf("open %q: %v", header.Name, err)
			}
			if _, err := io.Copy(outFile, tr); err != nil {
				outFile.Close()
				return fmt.Errorf("copying %q: %v", header.Name, err)
			}
			outFile.Close()
			if header.AccessTime.IsZero() {
				header.AccessTime = header.ModTime
			}
			if header.ChangeTime.IsZero() {
				header.ChangeTime = header.ModTime
			}
			if err := os.Chtimes(fp, header.AccessTime, header.ChangeTime); err != nil {
				return fmt.Errorf("chtime %q: %v", header.Name, err)
			}

		case tar.TypeSymlink:
			if err := fs.MkdirAll(filepath.Dir(filepath.Join(rb.OverlayUpperPath(), step.ToPath, header.Name)), 0755); err != nil && !os.IsExist(err) {
				return fmt.Errorf("mkdir %q: %v", header.Name, err)
			}
			if err := fs.Symlink(header.Linkname, filepath.Join(rb.OverlayUpperPath(), step.ToPath, header.Name)); err != nil {
				return fmt.Errorf("creating symlink for %q: %v", header.Name, err)
			}

		case tar.TypeXGlobalHeader:
			// Ignore.
		default:
			return fmt.Errorf("unsupported tar resource: %x", header.Typeflag)
		}
	}
	return nil
}
