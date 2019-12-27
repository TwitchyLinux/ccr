// Package deb works with debian packages.
package deb

import (
	"io"
	"os"

	"github.com/twitchylinux/ccr"
	"github.com/twitchyliquid64/debdep"
	"github.com/twitchyliquid64/debdep/deb"
)

func DebPackages(cache *ccr.Cache) (*debdep.PackageInfo, error) {
	pkgPath := cache.GetDebPkgsPath()

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
