package gen

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
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
func unpackedDeb(gc GenerationContext, src *vts.Puesdo) (*dpkg.Deb, error) {
	var dr cache.ReadSeekCloser
	var err error

	cv, ok := gc.Cache.GetObj(src.SHA256)
	if ok {
		return cv.(*dpkg.Deb), nil
	}

	if src.URL != "" {
		if dr, err = deb.PkgReader(gc.Cache, src.SHA256, src.URL); err != nil {
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
	gc.Cache.PutObj(src.SHA256, d)

	return d, nil
}

// GenerateDebSource implements generation of a resource target, based
// on a reference to a debian package as its source.
func GenerateDebSource(gc GenerationContext, resource *vts.Resource, src *vts.Puesdo) error {
	p, err := determinePath(resource, gc.RunnerEnv)
	if err != nil {
		return vts.WrapWithTarget(err, resource)
	}

	d, err := unpackedDeb(gc, src)
	if err != nil {
		return vts.WrapWithTarget(err, src)
	}

	// TODO: better way to do this?
	for _, f := range d.Files() {
		if f.Hdr.Name == "."+p {
			w, err := gc.RunnerEnv.FS.OpenFile(p, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, os.FileMode(f.Hdr.Mode))
			if err != nil {
				return vts.WrapWithTarget(vts.WrapWithPath(err, p), resource)
			}
			defer w.Close()
			if _, err := io.Copy(w, bytes.NewReader(f.Data)); err != nil {
				return vts.WrapWithTarget(vts.WrapWithPath(err, p), resource)
			}
			return nil
		}
	}

	return fmt.Errorf("couldnt find %s in deb", p)
}
