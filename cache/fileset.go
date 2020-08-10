package cache

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"io/ioutil"
	"os"

	"github.com/google/crfs/stargz"
)

// PendingFileset implements writing a set of files into the cache
// as a specific cache hash.
type PendingFileset struct {
	tmpFile *os.File
	f       *os.File
	gzip    *stargz.Writer
	tar     *tar.Writer
}

func (pfs *PendingFileset) Close() error {
	var err error
	if err2 := pfs.tar.Close(); err != nil {
		err = err2
	}
	if err2 := pfs.tmpFile.Sync(); err != nil {
		err = err2
	}
	if _, err2 := pfs.tmpFile.Seek(0, 0); err2 != nil {
		err = err2
	}

	if err2 := pfs.gzip.AppendTar(pfs.tmpFile); err2 != nil {
		err = err2
	}
	if err2 := pfs.gzip.Close(); err != nil {
		err = err2
	}
	if err2 := pfs.f.Close(); err != nil {
		err = err2
	}
	pfs.tmpFile.Close()
	os.Remove(pfs.tmpFile.Name())
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
	t, err := ioutil.TempFile("", "")
	if err != nil {
		return nil, err
	}
	return &PendingFileset{f: f, tmpFile: t, gzip: stargz.NewWriter(f), tar: tar.NewWriter(t)}, nil
}

func (c *Cache) FileInFileset(fsHash []byte, fsPath string) (io.Reader, io.Closer, os.FileMode, error) {
	f, err := c.ByHash(fsHash)
	if err != nil {
		return nil, nil, 0, err
	}

	osF, ok := f.(*os.File)
	if !ok {
		return nil, nil, 0, fmt.Errorf("expected reader to be *os.File, got %T", f)
	}
	s, err := osF.Stat()
	if err != nil {
		return nil, nil, 0, err
	}

	sgz, err := stargz.Open(io.NewSectionReader(f, 0, s.Size()))
	if err != nil {
		return nil, nil, 0, fmt.Errorf("reading gzip: %v", err)
	}

	e, ok := sgz.Lookup(fsPath)
	if !ok {
		f.Close()
		return nil, nil, 0, os.ErrNotExist
	}

	r, err := sgz.OpenFile(fsPath)
	if err != nil {
		f.Close()
		return nil, nil, 0, fmt.Errorf("reading file: %v", err)
	}

	return r, f, os.FileMode(e.Mode), nil
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
	if h.Name == "stargz.index.json" {
		return fsr.Next()
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
