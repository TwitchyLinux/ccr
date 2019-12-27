package main

import (
	"fmt"
	"strings"

	"github.com/twitchylinux/ccr/ccr/deb"
)

func goDebGenCmd(pkg string) error {
	pkgs, err := deb.DebPackages(cache)
	if err != nil {
		return err
	}

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

	return nil
}
