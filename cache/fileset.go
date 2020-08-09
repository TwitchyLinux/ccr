package cache

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
)

// PendingFileset implements writing a set of files into the cache
// as a specific cache hash.
type PendingFileset struct {
	f    *os.File
	gzip *gzip.Writer
	tar  *tar.Writer
}

func (pfs *PendingFileset) Close() error {
	var err error
	if err2 := pfs.tar.Close(); err != nil {
		err = err2
	}
	if err2 := pfs.gzip.Close(); err != nil {
		err = err2
	}
	if err2 := pfs.f.Close(); err != nil {
		err = err2
	}
	return err
}

func (pfs *PendingFileset) AddSymlink(path string, info os.FileInfo, target string) error {
	if err := pfs.tar.WriteHeader(&tar.Header{
		Typeflag: tar.TypeSymlink,
		Linkname: target,
		Name:     path,
		Size:     info.Size(),
		Mode:     int64(info.Mode()),
		ModTime:  info.ModTime(),
	}); err != nil {
		return fmt.Errorf("writing header: %v", err)
	}
	return nil
}

func (pfs *PendingFileset) AddFile(path string, info os.FileInfo, content io.ReadCloser) error {
	if err := pfs.tar.WriteHeader(&tar.Header{
		Name:    path,
		Size:    info.Size(),
		Mode:    int64(info.Mode()),
		ModTime: info.ModTime(),
	}); err != nil {
		content.Close()
		return fmt.Errorf("writing header: %v", err)
	}
	if _, err := io.Copy(pfs.tar, content); err != nil {
		content.Close()
		return fmt.Errorf("copy: %v", err)
	}
	if err := content.Close(); err != nil {
		return fmt.Errorf("close: %v", err)
	}
	return nil
}

func (c *Cache) CommitFileset(hash []byte) (*PendingFileset, error) {
	f, err := c.HashWriter(hash)
	if err != nil {
		return nil, err
	}
	gz := gzip.NewWriter(f)
	return &PendingFileset{f: f, gzip: gz, tar: tar.NewWriter(gz)}, nil
}

func (c *Cache) FileInFileset(fsHash []byte, fsPath string) (io.Reader, io.Closer, os.FileMode, error) {
	f, err := c.ByHash(fsHash)
	if err != nil {
		return nil, nil, 0, err
	}

	tape, err := gzip.NewReader(f)
	if err != nil {
		return nil, nil, 0, fmt.Errorf("reading gzip: %v", err)
	}
	tr := tar.NewReader(tape)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			f.Close()
			return nil, nil, 0, fmt.Errorf("reading tar: %v", err)
		}

		switch header.Typeflag {
		case tar.TypeDir:
		case tar.TypeReg:
			if header.Name == fsPath {
				return tr, f, os.FileMode(header.Mode), nil
			}
		default:
			f.Close()
			return nil, nil, 0, fmt.Errorf("unsupported tar resource: %x", header.Typeflag)
		}
	}

	f.Close()
	return nil, nil, 0, os.ErrNotExist
}

type FilesetReader struct {
	f    io.Closer
	tape *tar.Reader
}

func (fsr *FilesetReader) Close() error {
	return fsr.f.Close()
}

func (fsr *FilesetReader) Next() (path string, header *tar.Header, err error) {
	h, err := fsr.tape.Next()
	if err != nil {
		return "", nil, err
	}
	return h.Name, h, nil
}

func (fsr *FilesetReader) Read(b []byte) (int, error) {
	return fsr.tape.Read(b)
}

func (c *Cache) FilesetReader(fsHash []byte) (*FilesetReader, error) {
	f, err := c.ByHash(fsHash)
	if err != nil {
		return nil, err
	}

	tape, err := gzip.NewReader(f)
	if err != nil {
		return nil, fmt.Errorf("reading gzip: %v", err)
	}
	return &FilesetReader{
		f:    f,
		tape: tar.NewReader(tape),
	}, nil
}
