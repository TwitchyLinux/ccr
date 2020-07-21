// Package deb works with debian packages.
package deb

import (
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/twitchylinux/ccr/cache"
	"github.com/twitchyliquid64/debdep"
	"github.com/twitchyliquid64/debdep/deb"
)

const debInfoName = "deb-pkgs"

func DebPackages(cache *cache.Cache) (*debdep.PackageInfo, error) {
	pkgPath := cache.NamePath(debInfoName)

	// Write it out if it doesn't exist.
	if _, err := os.Stat(pkgPath); err != nil {
		d, err := debdep.RepositoryPackagesReader(debdep.ResolverConfig{
			Codename:     "stable",
			Distribution: "stable",
			Component:    "main",
			Arch: deb.Arch{
				Arch: "amd64",
			},
			BaseURL: "https://cdn-aws.deb.debian.org/debian",
		}, true)
		if err != nil {
			return nil, err
		}
		defer d.Close()
		f, err := os.OpenFile(pkgPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
		if err != nil {
			return nil, err
		}
		if _, err := io.Copy(f, d); err != nil {
			return nil, err
		}
		if err := f.Close(); err != nil {
			return nil, err
		}
	}

	return debdep.LoadPackageInfo(debdep.DefaultResolverConfig, pkgPath, true)
}

// PkgReader returns a reader to the deb package, caching and
// downloading it if necessary.
func PkgReader(c *cache.Cache, sha256, url string) (cache.ReadSeekCloser, error) {
	f, err := c.BySHA256(sha256)
	switch {
	case err == cache.ErrCacheMiss:
	case err == nil:
		return f, nil
	default:
		return nil, err
	}

	// Download.
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	r, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer r.Body.Close()

	switch r.StatusCode {
	case http.StatusOK:
	default:
		return nil, fmt.Errorf("unexpected response code '%d' (%s)", r.StatusCode, r.Status)
	}
	w, err := os.OpenFile(c.SHA256Path(sha256), os.O_CREATE|os.O_TRUNC|os.O_RDWR, 0644)
	if err != nil {
		return nil, err
	}
	if _, err := io.Copy(w, r.Body); err != nil {
		w.Close()
		return nil, err
	}
	w.Seek(0, os.SEEK_SET)
	return w, nil
}
