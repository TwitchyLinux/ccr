package main

import (
	"archive/tar"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/twitchylinux/ccr"
	"github.com/twitchylinux/ccr/ccr/buildgen"
	"github.com/twitchylinux/ccr/vts"
	"github.com/twitchylinux/ccr/vts/common"
)

func dumpFilesetContents(h []byte) error {
	fr, err := resCache.FilesetReader(h)
	if err != nil {
		return err
	}
	defer fr.Close()

	for {
		path, h, err := fr.Next()
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}

		size := fmt.Sprint(h.Size)
		size += strings.Repeat(" ", 10-len(size))

		switch h.Typeflag {
		case tar.TypeReg:
			fmt.Printf("%s %s %s\n", os.FileMode(h.Mode), size, path)
		case tar.TypeSymlink:
			fmt.Printf("%s %s %s  -->  %s\n", os.FileMode(h.Mode), size, path, h.Linkname)
		}
	}
	return nil
}

func doBuildgenCmd(target string) error {
	uv := ccr.NewUniverse(nil, resCache)

	dr := ccr.NewDirResolver(*dir)
	findOpts := ccr.FindOptions{
		FallbackResolvers: []ccr.CCRResolver{dr.Resolve},
		PrefixResolvers: map[string]ccr.CCRResolver{
			"common": common.Resolve,
		},
	}

	if err := uv.Build([]vts.TargetRef{{Path: target}}, &findOpts, *baseDir); err != nil {
		return err
	}
	t := uv.GetTarget(target)
	h, err := uv.TargetRollupHash(target)
	if err != nil {
		return err
	}
	fmt.Printf("[%x] %s\n", h, t)
	if err := uv.Generate(ccr.GenerateConfig{}, vts.TargetRef{Path: target}, *baseDir); err != nil {
		return err
	}

	if err := dumpFilesetContents(h); err != nil {
		return err
	}
	fmt.Printf("\n\n")

	bg, err := buildgen.New(resCache, target[strings.LastIndex(target, ":"):], h)
	if err != nil {
		return err
	}
	return bg.Build(os.Stdout)
}
