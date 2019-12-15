package main

import (
	"flag"
	"fmt"

	"github.com/twitchylinux/ccr"
	"github.com/twitchylinux/ccr/vts"
	"github.com/twitchylinux/ccr/vts/common"
)

func doCoverageCmd() error {
	uv := ccr.NewUniverse()

	dr := ccr.NewDirResolver(*dir)
	findOpts := ccr.FindOptions{
		FallbackResolvers: []ccr.CCRResolver{dr.Resolve},
		PrefixResolvers: map[string]ccr.CCRResolver{
			"common": common.Resolve,
		},
	}

	var targets []vts.TargetRef
	for _, arg := range flag.Args()[1:] {
		targets = append(targets, vts.TargetRef{Path: arg})
	}
	if err := uv.Build(targets, &findOpts); err != nil {
		return err
	}

	for _, p := range uv.EnumeratedTargets() {
		if p.GlobalPath() != "" {
			fmt.Println(p.GlobalPath())
		}
	}

	return nil
}
