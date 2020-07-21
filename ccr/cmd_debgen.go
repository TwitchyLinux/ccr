package main

import (
	"archive/tar"
	"fmt"
	"os"
	"strings"

	"github.com/twitchylinux/ccr/ccr/deb"
	"github.com/twitchyliquid64/debdep"
	"github.com/twitchyliquid64/debdep/dpkg"
)

func goDebGenCmd(mode, pkg string) error {
	pkgs, err := deb.DebPackages(resCache)
	if err != nil {
		return err
	}

	// TODO: Decompose into own functions.
	switch mode {
	case "gen", "generate":
		p, err := pkgs.FindLatest(pkg)
		if err != nil {
			return err
		}

		dr, err := deb.PkgReader(resCache, p.Values["SHA256"], debdep.DefaultResolverConfig.BaseURL+"/"+p.Values["Filename"])
		if err != nil {
			return err
		}
		defer dr.Close()
		d, err := dpkg.Open(dr)
		if err != nil {
			return err
		}
		if err := mkDebResources(p, d, os.Stdout); err != nil {
			return err
		}

		src, err := mkDebSource(debdep.DefaultResolverConfig.BaseURL, p)
		if err != nil {
			return err
		}
		fmt.Println(src.String())
		return nil
	case "gensrc", "gen-src", "generate-source":
		p, err := pkgs.FindLatest(pkg)
		if err != nil {
			return err
		}

		src, err := mkDebSource(debdep.DefaultResolverConfig.BaseURL, p)
		if err != nil {
			return err
		}
		fmt.Println(src.String())
		return nil

	case "", "show":
		p, err := pkgs.FindLatest(pkg)
		if err != nil {
			return err
		}
		fmt.Printf("\n -= %s =-\n", strings.ToUpper(p.Values["Package"]))
		fmt.Printf("Version: %s\n", p.Values["Version"])

		dr, err := deb.PkgReader(resCache, p.Values["SHA256"], debdep.DefaultResolverConfig.BaseURL+"/"+p.Values["Filename"])
		if err != nil {
			return err
		}
		defer dr.Close()
		d, err := dpkg.Open(dr)
		if err != nil {
			return err
		}

		for _, f := range d.Files() {
			switch f.Hdr.Typeflag {
			case tar.TypeReg:
				fmt.Printf("File [%#o]: %s\n", f.Hdr.Mode, f.Hdr.Name)
			case tar.TypeDir:
				fmt.Printf("Dir  [%#o]: %s\n", f.Hdr.Mode, f.Hdr.Name)
			case tar.TypeLink, tar.TypeSymlink:
				fmt.Printf("Link [%#o]: %s -> %s\n", f.Hdr.Mode, f.Hdr.Name, f.Hdr.Linkname)
			}
		}

		return nil
	}

	return fmt.Errorf("no such subcommand %q", mode)
}
