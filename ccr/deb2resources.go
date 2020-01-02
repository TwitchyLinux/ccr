package main

import (
	"archive/tar"
	"bytes"
	"debug/elf"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/twitchylinux/ccr/vts/common"
	d2 "github.com/twitchyliquid64/debdep/deb"
	"github.com/twitchyliquid64/debdep/dpkg"
)

var (
	// These path prefixes are ignored - no resources are generated from them.
	ignoredPrefixes = []string{
		"./usr/share/doc/",
		"./usr/share/lintian",
	}
	// These directories are assumed to already exist, so no resources are
	// generated from them.
	ignoredDirs = map[string]bool{
		"./":                          true,
		"./usr/":                      true,
		"./usr/lib/":                  true,
		"./usr/share/":                true,
		"./usr/lib/x86_64-linux-gnu/": true,
	}
)

type resKind uint8

// Valid resKind values.
const (
	ResDir resKind = 1 + iota
	ResStdSo
	ResFile
)

type debResource interface {
	ResourceKind() resKind
	ResourceName() string
}

type debDir tar.Header

func (d debDir) ResourceKind() resKind {
	return ResDir
}

func (d debDir) ResourceName() string {
	return "dir_" + strings.Replace(filepath.Base(d.Name), " ", "_", -1)
}

type debStdSo dpkg.DataFile

func (d *debStdSo) ResourceKind() resKind {
	return ResStdSo
}

func (d *debStdSo) ResourceName() string {
	spl := strings.Split(filepath.Base(d.Hdr.Name), ".")
	switch len(spl) {
	case 1:
		return "lib_" + spl[0]
	case 2:
		return "lib_" + spl[0] + "_" + spl[1]
	default:
		return "lib_" + spl[0] + "_" + strings.Join(spl[2:], ".")
	}
}

type debFile dpkg.DataFile

func (d *debFile) ResourceKind() resKind {
	return ResFile
}

func (d *debFile) ResourceName() string {
	return "f_" + strings.Replace(filepath.Base(d.Hdr.Name), " ", "_", -1)
}

func matchPrefix(set []string, input string) bool {
	for i := range set {
		if strings.HasPrefix(input, set[i]) {
			return true
		}
	}
	return false
}

func soInStandardDir(path string) bool {
	return strings.HasPrefix(path, "./usr/lib/x86_64-linux-gnu/") && strings.Contains(filepath.Base(path), ".so")
}

func stdSoResource(f dpkg.DataFile) (debResource, error) {
	binData, err := elf.NewFile(bytes.NewReader(f.Data))
	if err != nil {
		return nil, fmt.Errorf("%q: failed decoding ELF: %v", f.Hdr.Name, err)
	}
	if binData.Type != elf.ET_DYN {
		return nil, fmt.Errorf("%q: elf type is non-exec %v", f.Hdr.Name, binData.Type)
	}
	r := debStdSo(f)
	return &r, nil
}

// file2Resource processes information about a file within a debian package,
// returning information to populate a resource target if needed.
func file2Resource(f dpkg.DataFile) (debResource, error) {
	if matchPrefix(ignoredPrefixes, f.Hdr.Name) {
		return nil, nil
	}

	switch f.Hdr.Typeflag {
	case tar.TypeReg:
		if soInStandardDir(f.Hdr.Name) {
			return stdSoResource(f)
		}
		df := debFile(f)
		return &df, nil

	case tar.TypeDir:
		if ignoredDirs[f.Hdr.Name] {
			return nil, nil
		}
		return debDir(f.Hdr), nil
	case tar.TypeLink, tar.TypeSymlink:
		fmt.Printf("Link [%#o]: %s -> %s\n", f.Hdr.Mode, f.Hdr.Name, f.Hdr.Linkname)
	}

	return nil, nil
}

func mkDebResources(p *d2.Paragraph, d *dpkg.Deb, w io.Writer) error {
	var names []string
	exists := map[string]bool{}
	for _, f := range d.Files() {
		r, err := file2Resource(f)
		if err != nil {
			return fmt.Errorf("%s: %v", f.Hdr.Name, err)
		}
		if r == nil {
			continue
		}

		rn := r.ResourceName()
		for {
			if !exists[rn] {
				break
			}
			rn = "_" + rn
		}
		exists[rn] = true
		fmt.Fprintf(w, "resource(\n  name = '%s',\n", rn)
		names = append(names, rn)
		switch r.ResourceKind() {
		case ResDir:
			fmt.Fprintf(w, "  parent = '%s',\n", common.DirResourceClass.Path)
			fmt.Fprintf(w, "  path   = '%s',\n", r.(debDir).Name[1:])
		case ResStdSo:
			fmt.Fprintf(w, "  parent = '%s',\n", common.SysLibResourceClass.Path)
			fmt.Fprintf(w, "  path   = '%s',\n", r.(*debStdSo).Hdr.Name[1:])
		case ResFile:
			fmt.Fprintf(w, "  parent = '%s',\n", common.FileResourceClass.Path)
			fmt.Fprintf(w, "  path   = '%s',\n", r.(*debFile).Hdr.Name[1:])
		default:
			fmt.Println(r)
		}
		fmt.Fprintf(w, "  source = '%s_%s',\n", ":debsrc", p.Values["Package"])
		fmt.Fprintf(w, ")\n\n")
	}

	fmt.Fprintf(w, "component(\n  name = '%s',\n", p.Values["Package"])
	fmt.Fprintf(w, "  deps = [\n")
	for _, name := range names {
		fmt.Fprintf(w, "    '%s',\n", name)
	}
	fmt.Fprintf(w, "  ]\n)\n\n")

	return nil
}
