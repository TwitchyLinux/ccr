package main

import (
	"errors"
	"flag"
	"fmt"
	"strings"

	"github.com/twitchylinux/ccr"
	"github.com/twitchylinux/ccr/vts"
	"github.com/twitchylinux/ccr/vts/common"
	"go.starlark.net/starlark"
)

func doQueryCmd(targetAttr string) error {
	uv := ccr.NewUniverse(nil, nil)

	dr := ccr.NewDirResolver(*dir)
	findOpts := ccr.FindOptions{
		FallbackResolvers: []ccr.CCRResolver{dr.Resolve},
		PrefixResolvers: map[string]ccr.CCRResolver{
			"common": common.Resolve,
		},
	}

	if !strings.HasPrefix(targetAttr, "//") {
		return fmt.Errorf("%q: must provide absolute path", targetAttr)
	}
	pIdx := strings.Index(targetAttr, "%")
	if pIdx < 0 {
		return errors.New("no attribute specified")
	}
	p, attr := targetAttr[:pIdx], targetAttr[pIdx+1:]

	if err := uv.Build([]vts.TargetRef{{Path: p}}, &findOpts, *baseDir); err != nil {
		return err
	}

	var (
		val starlark.Value
		err error
	)
	switch flag.Arg(0) {
	case "query", "query-by-name":
		val, err = uv.QueryByName(*baseDir, p, attr)
	case "query-by-class":
		val, err = uv.QueryByClass(*baseDir, p, attr)
	}
	if err != nil {
		return err
	}

	s, _ := starlark.AsString(val)
	fmt.Println(s)
	return nil
}
