package main

import (
	"fmt"
	"io"

	"github.com/twitchylinux/ccr/vts/common"
	d2 "github.com/twitchyliquid64/debdep/deb"
	"github.com/twitchyliquid64/debdep/dpkg"
)

func mkUniqueName(exists map[string]bool, rn string) string {
	for {
		if !exists[rn] {
			break
		}
		rn = "_" + rn
	}
	return rn
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

		rn := mkUniqueName(exists, r.ResourceName())
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
