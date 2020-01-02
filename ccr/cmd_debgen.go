package main

import (
	"archive/tar"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/twitchylinux/ccr"
	"github.com/twitchylinux/ccr/ccr/deb"
	"github.com/twitchyliquid64/debdep"
	d2 "github.com/twitchyliquid64/debdep/deb"
	"github.com/twitchyliquid64/debdep/dpkg"
)

// debReader returns a reader to the deb package, downloading it
// if necessary.
func debReader(p *d2.Paragraph) (io.ReadCloser, error) {
	f, err := cache.BySHA256(p.Values["SHA256"])
	switch err {
	case ccr.ErrCacheMiss:
	case nil:
		return f, nil
	default:
		return nil, err
	}

	// Download.
	req, err := http.NewRequest(http.MethodGet, debdep.DefaultResolverConfig.BaseURL+"/"+p.Values["Filename"], nil)
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
	w, err := os.OpenFile(cache.SHA256Path(p.Values["SHA256"]), os.O_CREATE|os.O_TRUNC|os.O_RDWR, 0644)
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

func goDebGenCmd(mode, pkg string) error {
	pkgs, err := deb.DebPackages(cache)
	if err != nil {
		return err
	}

	switch mode {
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

		dr, err := debReader(p)
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
