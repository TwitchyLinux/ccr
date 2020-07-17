package main

import (
	"flag"

	"github.com/twitchylinux/ccr"
	"github.com/twitchylinux/ccr/vts"
	"github.com/twitchylinux/ccr/vts/common"
)

func doGenerateCmd() error {
	uv := ccr.NewUniverse(nil, nil)

	dr := ccr.NewDirResolver(*dir)
	findOpts := ccr.FindOptions{
		FallbackResolvers: []ccr.CCRResolver{dr.Resolve},
		PrefixResolvers: map[string]ccr.CCRResolver{
			"common": common.Resolve,
		},
	}

	if err := uv.Build([]vts.TargetRef{{Path: flag.Arg(1)}}, &findOpts, *baseDir); err != nil {
		return err
	}
	if err := uv.Generate(ccr.GenerateConfig{}, vts.TargetRef{Path: flag.Arg(1)}, *baseDir); err != nil {
		return err
	}

	return nil
}
