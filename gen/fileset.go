package gen

import (
	"archive/tar"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/twitchylinux/ccr/proc"
	"github.com/twitchylinux/ccr/vts"
	"gopkg.in/src-d/go-billy.v4"
)

// fileset describes any set of files used as the input or output of
// a populate or generate operation.
type fileset interface {
	Close() error
	Next() (path string, header *tar.Header, err error)
	Read(b []byte) (int, error)
}

// unaryFileset implements the fileset interface for a single file.
type unaryFileset struct {
	f io.ReadCloser

	srcPath   string
	srcHeader tar.Header
}

func (fs *unaryFileset) Close() error {
	if fs.f != nil {
		if err := fs.f.Close(); err != nil {
			return err
		}
		fs.f = nil
	}
	return nil
}

func (fs *unaryFileset) Next() (path string, header *tar.Header, err error) {
	if fs.f == nil {
		var err error
		if fs.f, err = os.Open(fs.srcPath); err != nil {
			return "", nil, err
		}
		return filepath.Base(fs.srcPath), &fs.srcHeader, nil
	}

	// There's only one file in this fileset, so return EOF otherwise.
	return "", nil, io.EOF
}

func (fs *unaryFileset) Read(b []byte) (int, error) {
	if fs.f == nil {
		return 0, errors.New("file not open")
	}
	return fs.f.Read(b)
}

// filesetForSource returns a fileset which can be used to access the files
// provided by the given source.
func filesetForSource(gc GenerationContext, source vts.Target) (fileset, error) {
	switch src := source.(type) {
	case *vts.Puesdo:
		switch src.Kind {
		case vts.FileRef:
			return filesetForFileSource(src)
		case vts.DebRef:
			return filesetForDebSource(gc, src)
		}
		return nil, fmt.Errorf("cannot generate using puesdo source %v", src.Kind)

	case *vts.Generator:
		return nil, errors.New("not implemented")

	case *vts.Build:
		h, err := src.RollupHash(gc.RunnerEnv, proc.EvalComputedAttribute)
		if err != nil {
			return nil, vts.WrapWithTarget(err, src)
		}
		return gc.Cache.FilesetReader(h)

	case *vts.Sieve:
		return filesetForSieve(gc, src)
	}
	return nil, fmt.Errorf("cannot obtain fileset for source %T", source)
}

type skipFunc func(path string, hdr *tar.Header) (bool, error)

// populateFileToPath iterates through the provided fileset, writing the
// first file for which skip() returns false (or the first file if no
// skip function is provided) to the given path and with the given mode.
// If the provided mode is zero, the mode is read instead from the fileset.
func populateFileToPath(fs billy.Filesystem, src fileset, outPath string, mode os.FileMode, skip skipFunc) error {
	for {
		p, h, err := src.Next()
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}
		if mode == 0 {
			mode = os.FileMode(h.Mode)
		}

		if skip != nil {
			shouldSkip, err := skip(p, h)
			if err != nil {
				return err
			}
			if shouldSkip {
				continue
			}
		}

		w, err := fs.OpenFile(outPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, mode)
		if err != nil {
			return vts.WrapWithPath(err, outPath)
		}
		if _, err := io.Copy(w, src); err != nil {
			return vts.WrapWithPath(err, outPath)
		}
		return w.Close()
	}

	return os.ErrNotExist
}
