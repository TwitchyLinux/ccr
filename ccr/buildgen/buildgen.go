package buildgen

import (
	"archive/tar"
	"bytes"
	"flag"
	"fmt"
	"io"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/gobwas/glob"
	"github.com/twitchylinux/ccr/cache"
	"github.com/twitchylinux/ccr/vts/common"
)

var supportResourceFlag = flag.String("buildgen-support-resources", "", "comma-separated list of names to match path")

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

type supportResourceMatcher struct {
	name string
	p    glob.Glob
}

func (m *supportResourceMatcher) emit(b *Builder, path string, h *tar.Header) {
	resName := mkLibName(path, m.name)
	fmt.Fprintf(&b.supportRes, "resource(\n")
	fmt.Fprintf(&b.supportRes, "  name   = %q,\n", resName)
	fmt.Fprintf(&b.supportRes, "  parent = %s,\n", strconv.Quote(common.FileResourceClass.Path))
	fmt.Fprintf(&b.supportRes, "  path   = %s,\n", strconv.Quote("/"+path))
	fmt.Fprintf(&b.supportRes, "  mode   = %s,\n", strconv.Quote(fmt.Sprintf("%04o", h.Mode&0777)))
	fmt.Fprintf(&b.supportRes, "  source = %s,\n", strconv.Quote(b.target))
	fmt.Fprintf(&b.supportRes, ")\n\n")
	b.supportTargets = append(b.supportTargets, resName)
}

type Builder struct {
	cache        *cache.Cache
	target       string
	hash         []byte
	supportMatch []supportResourceMatcher

	devRes     bytes.Buffer
	libRes     bytes.Buffer
	binRes     bytes.Buffer
	supportRes bytes.Buffer

	headerDirs map[string]struct{}

	devTargets     []string
	libTargets     []string
	binTargets     []string
	supportTargets []string
}

func (b *Builder) header(path string, h *tar.Header) {
	if filepath.Dir(path) != "usr/include" { // directory
		relPath, err := filepath.Rel("usr/include", path)
		if err != nil {
			panic(err)
		}
		b.headerDirs[filepath.Dir(relPath)] = struct{}{}
		return
	}
	b.emitHeader(path, h)
}

func (b *Builder) emitHeader(path string, h *tar.Header) {
	resName := mkLibName(path, "h")
	fmt.Fprintf(&b.devRes, "resource(\n")
	fmt.Fprintf(&b.devRes, "  name   = %q,\n", resName)
	fmt.Fprintf(&b.devRes, "  parent = %s,\n", strconv.Quote(common.CHeaderResourceClass.Path))
	fmt.Fprintf(&b.devRes, "  path   = %s,\n", strconv.Quote("/"+path))
	fmt.Fprintf(&b.devRes, "  mode   = %s,\n", strconv.Quote(fmt.Sprintf("%04o", h.Mode&0777)))
	fmt.Fprintf(&b.devRes, "  source = %s,\n", strconv.Quote(b.target))
	fmt.Fprintf(&b.devRes, ")\n\n")
	b.devTargets = append(b.devTargets, resName)
}

func (b *Builder) emitPkgConfig(path string, h *tar.Header) {
	resName := mkLibName(path, "pkgconfig")
	fmt.Fprintf(&b.devRes, "resource(\n")
	fmt.Fprintf(&b.devRes, "  name   = %q,\n", resName)
	fmt.Fprintf(&b.devRes, "  parent = %s,\n", strconv.Quote(common.PkgcfgResourceClass.Path))
	fmt.Fprintf(&b.devRes, "  path   = %s,\n", strconv.Quote("/"+path))
	fmt.Fprintf(&b.devRes, "  mode   = %s,\n", strconv.Quote(fmt.Sprintf("%04o", h.Mode&0777)))
	fmt.Fprintf(&b.devRes, "  source = %s,\n", strconv.Quote(b.target))
	fmt.Fprintf(&b.devRes, ")\n\n")
	b.devTargets = append(b.devTargets, resName)
}

func (b *Builder) emitLibtoolDesc(path string, h *tar.Header) {
	resName := mkLibName(path, "la")
	fmt.Fprintf(&b.devRes, "resource(\n")
	fmt.Fprintf(&b.devRes, "  name   = %q,\n", resName)
	fmt.Fprintf(&b.devRes, "  parent = %s,\n", strconv.Quote(common.LibtoolDescResourceClass.Path))
	fmt.Fprintf(&b.devRes, "  path   = %s,\n", strconv.Quote("/"+path))
	fmt.Fprintf(&b.devRes, "  mode   = %s,\n", strconv.Quote(fmt.Sprintf("%04o", h.Mode&0777)))
	fmt.Fprintf(&b.devRes, "  source = %s,\n", strconv.Quote(b.target))
	fmt.Fprintf(&b.devRes, ")\n\n")
	b.devTargets = append(b.devTargets, resName)
}

func (b *Builder) emitStaticLib(path string, h *tar.Header) {
	resName := mkLibName(path, "static")
	fmt.Fprintf(&b.devRes, "resource(\n")
	fmt.Fprintf(&b.devRes, "  name   = %q,\n", resName)
	fmt.Fprintf(&b.devRes, "  parent = %s,\n", strconv.Quote(common.StaticLibResourceClass.Path))
	fmt.Fprintf(&b.devRes, "  path   = %s,\n", strconv.Quote("/"+path))
	fmt.Fprintf(&b.devRes, "  mode   = %s,\n", strconv.Quote(fmt.Sprintf("%04o", h.Mode&0777)))
	fmt.Fprintf(&b.devRes, "  source = %s,\n", strconv.Quote(b.target))
	fmt.Fprintf(&b.devRes, ")\n\n")
	b.devTargets = append(b.devTargets, resName)
}

func (b *Builder) emitLibSymlink(path string, h *tar.Header) {
	resName := mkLibName(path, "symlink")
	fmt.Fprintf(&b.libRes, "resource(\n")
	fmt.Fprintf(&b.libRes, "  name   = %q,\n", resName)
	fmt.Fprintf(&b.libRes, "  parent = %s,\n", strconv.Quote(common.SysLibLinkResourceClass.Path))
	fmt.Fprintf(&b.libRes, "  path   = %s,\n", strconv.Quote("/"+path))
	fmt.Fprintf(&b.libRes, "  target = %s,\n", strconv.Quote(h.Linkname))
	fmt.Fprintf(&b.libRes, "  source = %s,\n", strconv.Quote("common://generators:symlink"))
	fmt.Fprintf(&b.libRes, ")\n\n")
	b.libTargets = append(b.libTargets, resName)
}

func (b *Builder) emitSharedLib(path string, h *tar.Header) {
	resName := mkLibName(path, "")
	fmt.Fprintf(&b.libRes, "resource(\n")
	fmt.Fprintf(&b.libRes, "  name   = %q,\n", resName)
	fmt.Fprintf(&b.libRes, "  parent = %s,\n", strconv.Quote(common.SysLibResourceClass.Path))
	fmt.Fprintf(&b.libRes, "  path   = %s,\n", strconv.Quote("/"+path))
	fmt.Fprintf(&b.libRes, "  mode   = %s,\n", strconv.Quote(fmt.Sprintf("%04o", h.Mode&0777)))
	fmt.Fprintf(&b.libRes, "  source = %s,\n", strconv.Quote(b.target))
	fmt.Fprintf(&b.libRes, ")\n\n")
	b.libTargets = append(b.libTargets, resName)
}

func (b *Builder) emitBin(path string, h *tar.Header, kind string) {
	resName := mkLibName(path, kind)
	fmt.Fprintf(&b.binRes, "resource(\n")
	fmt.Fprintf(&b.binRes, "  name   = %q,\n", resName)
	if h.Typeflag == tar.TypeSymlink {
		fmt.Fprintf(&b.binRes, "  parent = %s,\n", strconv.Quote(common.BinLinkResourceClass.Path))
	} else {
		fmt.Fprintf(&b.binRes, "  parent = %s,\n", strconv.Quote(common.BinResourceClass.Path))
	}
	fmt.Fprintf(&b.binRes, "  path   = %s,\n", strconv.Quote("/"+path))
	if h.Typeflag == tar.TypeSymlink {
		fmt.Fprintf(&b.binRes, "  target = %s,\n", strconv.Quote(h.Linkname))
		fmt.Fprintf(&b.binRes, "  source = %s,\n", strconv.Quote("common://generators:symlink"))
	} else {
		fmt.Fprintf(&b.binRes, "  mode   = %s,\n", strconv.Quote(fmt.Sprintf("%04o", h.Mode&0777)))
		fmt.Fprintf(&b.binRes, "  source = %s,\n", strconv.Quote(b.target))
	}
	fmt.Fprintf(&b.binRes, ")\n\n")
	b.binTargets = append(b.binTargets, resName)
}

func (b *Builder) writeBinComponent(w io.Writer) {
	fmt.Fprintf(w, "component(\n")
	fmt.Fprintf(w, "  name   = %q,\n", "bins")
	fmt.Fprintf(w, "  deps   = [\n")
	for _, bin := range b.binTargets {
		fmt.Fprintf(w, "    %s,\n", strconv.Quote(":"+bin))
	}
	fmt.Fprintf(w, "  ],\n")
	fmt.Fprintf(w, ")\n\n")
}

func (b *Builder) writeLibComponent(w io.Writer) {
	fmt.Fprintf(w, "component(\n")
	fmt.Fprintf(w, "  name   = %q,\n", "libs")
	fmt.Fprintf(w, "  deps   = [\n")
	if len(b.supportTargets) > 0 {
		fmt.Fprintf(w, "    %s,\n", strconv.Quote(":support-files"))
	}
	for _, lib := range b.libTargets {
		fmt.Fprintf(w, "    %s,\n", strconv.Quote(":"+lib))
	}
	fmt.Fprintf(w, "  ],\n")
	fmt.Fprintf(w, ")\n\n")
}

func (b *Builder) writeDevComponent(w io.Writer) {
	fmt.Fprintf(w, "component(\n")
	fmt.Fprintf(w, "  name   = %q,\n", "dev")
	fmt.Fprintf(w, "  deps   = [\n")
	fmt.Fprintf(w, "    %s,\n", strconv.Quote(":libs"))
	for _, t := range b.devTargets {
		fmt.Fprintf(w, "    %s,\n", strconv.Quote(":"+t))
	}
	fmt.Fprintf(w, "  ],\n")
	fmt.Fprintf(w, ")\n\n")
}

func (b *Builder) writeSupportComponent(w io.Writer) {
	fmt.Fprintf(w, "component(\n")
	fmt.Fprintf(w, "  name   = %q,\n", "support-files")
	fmt.Fprintf(w, "  deps   = [\n")
	for _, t := range b.supportTargets {
		fmt.Fprintf(w, "    %s,\n", strconv.Quote(":"+t))
	}
	fmt.Fprintf(w, "  ],\n")
	fmt.Fprintf(w, ")\n\n")
}

func (b *Builder) finalize(w io.Writer) error {
	var sorted []string
	for basePath, _ := range b.headerDirs {
		sorted = append(sorted, basePath)
	}
	sort.Strings(sorted)
	for _, basePath := range sorted {
		b.emitHeaderDir(filepath.Join("usr/include", basePath), basePath)
	}
	return nil
}

func (b *Builder) emitHeaderDir(path, basePath string) {
	resName := strings.Replace(basePath, string(filepath.Separator), "-", -1) + "-h-dir"
	fmt.Fprintf(&b.devRes, "resource(\n")
	fmt.Fprintf(&b.devRes, "  name   = %q,\n", resName)
	fmt.Fprintf(&b.devRes, "  parent = %s,\n", strconv.Quote(common.CHeadersResourceClass.Path))
	fmt.Fprintf(&b.devRes, "  path   = %s,\n", strconv.Quote("/"+path))
	fmt.Fprintf(&b.devRes, "  source = sieve_prefix(%s, %q),\n", strconv.Quote(b.target), path)
	fmt.Fprintf(&b.devRes, ")\n\n")
	b.devTargets = append(b.devTargets, resName)
}

func New(c *cache.Cache, target string, h []byte) (*Builder, error) {
	var matchers []supportResourceMatcher
	for _, res := range strings.Split(*supportResourceFlag, ",") {
		eq := strings.Index(res, "=")
		if eq < 0 {
			continue
		}

		p, err := glob.Compile(strings.TrimPrefix(res[eq+1:], "/"))
		if err != nil {
			return nil, fmt.Errorf("invalid match pattern %q: %v", res[eq+1:], err)
		}
		matchers = append(matchers, supportResourceMatcher{
			name: res[:eq],
			p:    p,
		})
	}

	return &Builder{
		cache:        c,
		target:       target,
		hash:         h,
		supportMatch: matchers,
		headerDirs:   map[string]struct{}{},
	}, nil
}

func (b *Builder) Build(out io.Writer) error {
	fr, err := b.cache.FilesetReader(b.hash)
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

		for _, m := range b.supportMatch {
			if m.p.Match(path) {
				m.emit(b, path, h)
				continue
			}
		}

		switch {
		case strings.HasPrefix(path, "usr/bin/"):
			b.emitBin(path, h, "bin")
		case strings.HasPrefix(path, "sbin/"):
			b.emitBin(path, h, "sbin")

		case strings.HasPrefix(path, "usr/include/") && strings.HasSuffix(path, ".h"):
			b.header(path, h)

		case (strings.HasPrefix(path, "usr/lib/pkgconfig") || strings.HasPrefix(path, "lib/pkgconfig")) && strings.HasSuffix(path, ".pc"):
			b.emitPkgConfig(path, h)

		case (strings.HasPrefix(path, "usr/lib/") || strings.HasPrefix(path, "lib/")) && strings.HasSuffix(path, ".la"):
			b.emitLibtoolDesc(path, h)

		case (strings.HasPrefix(path, "usr/lib/") || strings.HasPrefix(path, "lib/")) && strings.HasSuffix(path, ".a"):
			b.emitStaticLib(path, h)

		case (strings.HasPrefix(path, "usr/lib/") || strings.HasPrefix(path, "lib/")) &&
			(strings.HasPrefix(filepath.Base(path), "lib") || strings.Contains(filepath.Base(path), ".so.") || strings.HasSuffix(filepath.Base(path), ".so")):
			if bp := filepath.Dir(path); bp == "lib" || bp == "usr/lib" {
				switch h.Typeflag {
				case tar.TypeReg:
					b.emitSharedLib(path, h)

				case tar.TypeSymlink:
					b.emitLibSymlink(path, h)
				}
			}
		}
	}

	if err := b.finalize(out); err != nil {
		return err
	}
	if len(b.binTargets) > 0 {
		b.writeBinComponent(out)
		io.Copy(out, &b.binRes)
	}
	b.writeLibComponent(out)
	io.Copy(out, &b.libRes)
	if len(b.supportTargets) > 0 {
		b.writeSupportComponent(out)
		io.Copy(out, &b.supportRes)
	}
	b.writeDevComponent(out)
	io.Copy(out, &b.devRes)
	return nil
}
