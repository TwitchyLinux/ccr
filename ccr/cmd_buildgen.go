package main

import (
	"archive/tar"
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/twitchylinux/ccr"
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

func mkLibName(path, detail string) string {
	spl := strings.Split(filepath.Base(path), ".")
	if len(spl) <= 2 {
		if detail == "" {
			return spl[0]
		}
		return spl[0] + "-" + detail
	}

	if detail == "" {
		return spl[0] + "-" + strings.Join(spl[2:], "-")
	}
	return spl[0] + "-" + detail + "-" + strings.Join(spl[2:], "-")
}

func generateFromBuild(h []byte, target string) error {
	fr, err := resCache.FilesetReader(h)
	if err != nil {
		return err
	}
	defer fr.Close()

	var (
		b          bytes.Buffer
		devTargets []string
		libTargets []string
		w          io.Writer = &b
	)
	for {
		path, h, err := fr.Next()
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}

		switch {
		case strings.HasPrefix(path, "usr/include/") && strings.HasSuffix(path, ".h"):
			fmt.Fprintf(w, "resource(\n")
			fmt.Fprintf(w, "  name   = %q,\n", mkLibName(path, "h"))
			fmt.Fprintf(w, "  parent = %s,\n", strconv.Quote(common.CHeaderResourceClass.Path))
			fmt.Fprintf(w, "  path   = %s,\n", strconv.Quote("/"+path))
			fmt.Fprintf(w, "  mode   = %s,\n", strconv.Quote(fmt.Sprintf("%04o", h.Mode)))
			fmt.Fprintf(w, "  source = %s,\n", strconv.Quote(target))
			fmt.Fprintf(w, ")\n\n")
			devTargets = append(devTargets, mkLibName(path, "h"))

		case strings.HasPrefix(path, "usr/lib/pkgconfig") && strings.HasSuffix(path, ".pc"):
			fmt.Fprintf(w, "resource(\n")
			fmt.Fprintf(w, "  name   = %q,\n", mkLibName(path, "pkgconfig"))
			fmt.Fprintf(w, "  parent = %s,\n", strconv.Quote(common.PkgcfgResourceClass.Path))
			fmt.Fprintf(w, "  path   = %s,\n", strconv.Quote("/"+path))
			fmt.Fprintf(w, "  mode   = %s,\n", strconv.Quote(fmt.Sprintf("%04o", h.Mode)))
			fmt.Fprintf(w, "  source = %s,\n", strconv.Quote(target))
			fmt.Fprintf(w, ")\n\n")
			devTargets = append(devTargets, mkLibName(path, "pkgconfig"))

		case strings.HasPrefix(path, "usr/lib/") && strings.HasSuffix(path, ".la"):
			fmt.Fprintf(w, "resource(\n")
			fmt.Fprintf(w, "  name   = %q,\n", mkLibName(path, "la"))
			fmt.Fprintf(w, "  parent = %s,\n", strconv.Quote(common.LibtoolDescResourceClass.Path))
			fmt.Fprintf(w, "  path   = %s,\n", strconv.Quote("/"+path))
			fmt.Fprintf(w, "  mode   = %s,\n", strconv.Quote(fmt.Sprintf("%04o", h.Mode)))
			fmt.Fprintf(w, "  source = %s,\n", strconv.Quote(target))
			fmt.Fprintf(w, ")\n\n")
			devTargets = append(devTargets, mkLibName(path, "la"))

		case strings.HasPrefix(path, "usr/lib/") && strings.HasSuffix(path, ".a"):
			fmt.Fprintf(w, "resource(\n")
			fmt.Fprintf(w, "  name   = %q,\n", mkLibName(path, "static"))
			fmt.Fprintf(w, "  parent = %s,\n", strconv.Quote(common.StaticLibResourceClass.Path))
			fmt.Fprintf(w, "  path   = %s,\n", strconv.Quote("/"+path))
			fmt.Fprintf(w, "  mode   = %s,\n", strconv.Quote(fmt.Sprintf("%04o", h.Mode)))
			fmt.Fprintf(w, "  source = %s,\n", strconv.Quote(target))
			fmt.Fprintf(w, ")\n\n")
			devTargets = append(devTargets, mkLibName(path, "static"))

		case strings.HasPrefix(path, "usr/lib/") && strings.HasPrefix(filepath.Base(path), "lib"):
			switch h.Typeflag {
			case tar.TypeReg:
				if spl := strings.Split(filepath.Base(path), "."); len(spl) > 1 && spl[1] == "so" {
					fmt.Fprintf(w, "resource(\n")
					fmt.Fprintf(w, "  name   = %q,\n", mkLibName(path, ""))
					fmt.Fprintf(w, "  parent = %s,\n", strconv.Quote(common.SysLibResourceClass.Path))
					fmt.Fprintf(w, "  path   = %s,\n", strconv.Quote("/"+path))
					fmt.Fprintf(w, "  mode   = %s,\n", strconv.Quote(fmt.Sprintf("%04o", h.Mode)))
					fmt.Fprintf(w, "  source = %s,\n", strconv.Quote(target))
					fmt.Fprintf(w, ")\n\n")
					libTargets = append(libTargets, mkLibName(path, ""))
				}

			case tar.TypeSymlink:
				fmt.Fprintf(w, "resource(\n")
				fmt.Fprintf(w, "  name   = %q,\n", mkLibName(path, "symlink"))
				fmt.Fprintf(w, "  parent = %s,\n", strconv.Quote(common.SysLibLinkResourceClass.Path))
				fmt.Fprintf(w, "  path   = %s,\n", strconv.Quote("/"+path))
				fmt.Fprintf(w, "  target = %s,\n", strconv.Quote(h.Linkname))
				fmt.Fprintf(w, "  source = %s,\n", strconv.Quote("common://generators:symlink"))
				fmt.Fprintf(w, ")\n\n")
				libTargets = append(libTargets, mkLibName(path, "symlink"))
			}
		}
	}

	w = os.Stdout
	fmt.Fprintf(w, "component(\n")
	fmt.Fprintf(w, "  name   = %q,\n", "libs")
	fmt.Fprintf(w, "  deps   = [\n")
	for _, lib := range libTargets {
		fmt.Fprintf(w, "    %s,\n", strconv.Quote(":"+lib))
	}
	fmt.Fprintf(w, "  ],\n")
	fmt.Fprintf(w, ")\n\n")
	fmt.Fprintf(w, "component(\n")
	fmt.Fprintf(w, "  name   = %q,\n", "dev")
	fmt.Fprintf(w, "  deps   = [\n")
	fmt.Fprintf(w, "    %s,\n", strconv.Quote(":libs"))
	for _, t := range devTargets {
		fmt.Fprintf(w, "    %s,\n", strconv.Quote(":"+t))
	}
	fmt.Fprintf(w, "  ],\n")
	fmt.Fprintf(w, ")\n\n")

	_, err = io.Copy(os.Stdout, &b)
	return err
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

	td, err := ioutil.TempDir("", "")
	if err != nil {
		return err
	}
	defer os.RemoveAll(td)

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
	if err := generateFromBuild(h, target[strings.LastIndex(target, ":"):]); err != nil {
		return err
	}

	return nil
}
