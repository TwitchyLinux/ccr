package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/twitchylinux/ccr"
	"github.com/twitchylinux/ccr/ccr/deb"
	"github.com/twitchyliquid64/debdep"
	d2 "github.com/twitchyliquid64/debdep/deb"
)

// debReader returns a reader to the deb package, downloading it
// if necessary.
func debReader(p *d2.Paragraph) (io.ReadCloser, error) {
	f, err := cache.BySHA256(p.Values["SHA256"])
	if err != nil && err != ccr.ErrCacheMiss {
		return nil, err
	}
	if err == nil {
		return f, nil
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
	case "", "show":
		p, err := pkgs.FindLatest(pkg)
		if err != nil {
			return err
		}
		fmt.Printf("\n -= %s =-\n", strings.ToUpper(p.Values["Package"]))
		fmt.Printf("Breaks:  %s\n", p.Values["Breaks"])
		fmt.Printf("Depends: %s\n", p.Values["Depends"])
		if _, hasPre := p.Values["Pre-Depends"]; hasPre {
			fmt.Printf("Pre-Depends: %s\n", p.Values["Pre-Depends"])
		}
		fmt.Printf("Version: %s\n", p.Values["Version"])

		dr, err := debReader(p)
		if err != nil {
			return err
		}
		fmt.Println(dr)

		return nil
	}

	return fmt.Errorf("no such subcommand %q", mode)
}
