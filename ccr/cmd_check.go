package main

import (
	"flag"

	"github.com/twitchylinux/ccr"
	"github.com/twitchylinux/ccr/vts"
	"github.com/twitchylinux/ccr/vts/common"
)

func doCheckCmd() error {
	uv := ccr.NewUniverse(nil, nil)

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
	if err := uv.Build(targets, &findOpts, *baseDir); err != nil {
		return err
	}
	if err := uv.Check(targets, *baseDir); err != nil {
		return err
	}

	return nil
}
