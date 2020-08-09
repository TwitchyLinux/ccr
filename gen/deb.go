package gen

import (
	"archive/tar"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/twitchylinux/ccr/cache"
	"github.com/twitchylinux/ccr/ccr/deb"
	"github.com/twitchylinux/ccr/vts"
	"github.com/twitchyliquid64/debdep/dpkg"
)

// unpackedDeb resolves a debian package referenced in a target,
// to a *dpkg.Deb object.
func unpackedDeb(c *cache.Cache, src *vts.Puesdo) (*dpkg.Deb, error) {
	var dr cache.ReadSeekCloser
	var err error

	cv, ok := c.GetObj(src.SHA256)
	if ok {
		return cv.(*dpkg.Deb), nil
	}

	if src.URL != "" {
		s256, err := hex.DecodeString(src.SHA256)
		if err != nil {
			return nil, err
		}
		if dr, err = deb.PkgReader(c, s256, src.URL); err != nil {
			return nil, err
		}
	} else {
		if dr, err = os.Open(filepath.Join(filepath.Dir(src.ContractPath), src.Path)); err != nil {
			return nil, err
		}
	}
	defer dr.Close()

	// Verify hash.
	hasher := sha256.New()
	if _, err := io.Copy(hasher, dr); err != nil {
		return nil, err
	}
	if got, want := strings.ToLower(hex.EncodeToString(hasher.Sum(nil))), strings.ToLower(src.SHA256); got != want {
		return nil, fmt.Errorf("sha256 mismatch: got %s but expected %s", got, want)
	}

	var d *dpkg.Deb
	dr.Seek(0, os.SEEK_SET)
	if d, err = dpkg.Open(dr); err != nil {
		return nil, fmt.Errorf("failed decoding deb: %v", err)
	}
	c.PutObj(src.SHA256, d)

	return d, nil
}

// debFileset implements the fileset interface for files in a debian package.
type debFileset struct {
	files []dpkg.DataFile
	r     *bytes.Reader
}

func (fs *debFileset) Close() error {
	fs.files, fs.r = nil, nil
	return nil
}

func (fs *debFileset) Next() (path string, header *tar.Header, err error) {
	if len(fs.files) == 0 {
		return "", nil, io.EOF
	}
	file := fs.files[0]
	fs.files = fs.files[1:]

	fs.r = bytes.NewReader(file.Data)
	file.Hdr.Name = strings.TrimPrefix(file.Hdr.Name, ".")
	return file.Hdr.Name, &file.Hdr, nil
}

func (fs *debFileset) Read(b []byte) (int, error) {
	if fs.r == nil {
		return 0, errors.New("file not open")
	}
	return fs.r.Read(b)
}

func filesetForDebSource(gc GenerationContext, src *vts.Puesdo) (*debFileset, error) {
	d, err := unpackedDeb(gc.Cache, src)
	if err != nil {
		return nil, vts.WrapWithTarget(err, src)
	}
	return &debFileset{files: d.Files()}, nil
}
