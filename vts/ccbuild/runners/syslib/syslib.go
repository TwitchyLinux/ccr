// Package syslib implements a global check of runtime link dependencies.
package syslib

import (
	"crypto/sha256"
	"debug/elf"
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/twitchylinux/ccr/vts"
	"github.com/twitchylinux/ccr/vts/ccbuild/info"
	"go.starlark.net/starlark"
)

var systemTrustedDirs = []string{"/lib", "/usr/lib"}

// RuntimeLinkChecker returns a global runner that can check runtime
// link dependencies are satisfied.
func RuntimeLinkChecker() *globalChecker {
	return &globalChecker{
		libsInDirCache: make(map[string]map[string]*vts.Resource, 16),
	}
}

type globalChecker struct {
	libsInDirCache map[string]map[string]*vts.Resource
	libDirs        map[string]*vts.Resource
}

func (*globalChecker) Kind() vts.CheckerKind { return vts.ChkKindGlobal }

func (*globalChecker) String() string { return "syslib.link_checker" }

func (*globalChecker) Freeze() {}

func (*globalChecker) Truth() starlark.Bool { return true }

func (*globalChecker) Type() string { return "runner" }

func (t *globalChecker) Hash() (uint32, error) {
	h := sha256.Sum256([]byte(fmt.Sprintf("%p", t)))
	return uint32(uint32(h[0]) + uint32(h[1])<<8 + uint32(h[2])<<16 + uint32(h[3])<<24), nil
}

func (c *globalChecker) Run(chkr *vts.Checker, opts *vts.RunnerEnv) error {
	var err error

	if c.libDirs, err = getLibraryDirs(opts); err != nil {
		return fmt.Errorf("enumerating system library dirs: %v", err)
	}
	if len(c.libDirs) == 0 {
		return errors.New("no system libraries declared in universe")
	}

	bins, err := getBinaries(opts)
	if err != nil {
		return fmt.Errorf("enumerating binaries: %v", err)
	}
	if len(bins) == 0 {
		return errors.New("no binaries declared in universe")
	}
	for path, bin := range bins {
		if err := c.checkELFFile(path, bin, opts); err != nil {
			return vts.WrapWithPath(vts.WrapWithTarget(err, bin), path)
		}
	}

	return nil
}

func (*globalChecker) elfInfo(r *vts.Resource, opts *vts.RunnerEnv) (elf.FileHeader, info.ELFLinkDeps,
	[]info.ELFSym, string, error) {
	if !r.RuntimeInfo().HasRun(info.ELFPopulator) {
		if err := info.ELFPopulator.Run(r, opts, r.RuntimeInfo()); err != nil {
			return elf.FileHeader{}, info.ELFLinkDeps{}, nil, "", err
		}
	}
	d, err := r.RuntimeInfo().Get(info.ELFPopulator, info.ELFHeader)
	if err != nil {
		return elf.FileHeader{}, info.ELFLinkDeps{}, nil, "", err
	}
	elfHeader := d.(elf.FileHeader)
	if d, err = r.RuntimeInfo().Get(info.ELFPopulator, info.ELFDynamicSymbols); err != nil {
		return elf.FileHeader{}, info.ELFLinkDeps{}, nil, "", err
	}
	syms := d.([]info.ELFSym)
	if d, err = r.RuntimeInfo().Get(info.ELFPopulator, info.ELFDeps); err != nil {
		return elf.FileHeader{}, info.ELFLinkDeps{}, nil, "", err
	}
	deps := d.(info.ELFLinkDeps)
	if d, err = r.RuntimeInfo().Get(info.ELFPopulator, info.ELFInterpreter); err != nil {
		return elf.FileHeader{}, info.ELFLinkDeps{}, nil, "", err
	}
	interp := d.(string)
	return elfHeader, deps, syms, interp, nil
}

func (c *globalChecker) checkInterp(interp string, opts *vts.RunnerEnv) error {
	if interp == "" {
		return nil // No linker declared - no worries.
	}

	t, err := opts.Universe.FindByPath(interp, opts)
	if err != nil {
		return fmt.Errorf("couldnt read resource representing declared dynamic linker (%q): %v", interp, err)
	}
	r, ok := t.(*vts.Resource)
	if !ok {
		return vts.WrapWithTarget(fmt.Errorf("interpreter %q is a %s, not a resource", interp, t.TargetType().String()), t)
	}
	if p := r.Parent.Target.(*vts.ResourceClass).Path; p != "common://resources:sys_library" && p != "common://resources:sys_library_symlink" {
		return vts.WrapWithTarget(fmt.Errorf("interpreter %q is of class %q, need sys_library*", interp, p), t)
	}
	// Because it was of class sys_library, the ELF checkers on sys_library would
	// have validated the correctness of the ELF markup. As such we are done.
	return nil
}

func (c *globalChecker) expandPathStr(elfPath, pathStr string) (string, error) {
	pathStr = strings.Replace(pathStr, "$ORIGIN", filepath.Dir(elfPath), -1)
	pathStr = strings.Replace(pathStr, "${ORIGIN}", filepath.Dir(elfPath), -1)
	pathStr = strings.TrimSuffix(pathStr, "/.")

	if strings.Contains(pathStr, "$LIB") || strings.Contains(pathStr, "${LIB}") {
		return "", fmt.Errorf("%q: $LIB expansion not yet supported", pathStr)
	}
	if strings.Contains(pathStr, "$PLATFORM") || strings.Contains(pathStr, "${PLATFORM}") {
		return "", fmt.Errorf("%q: $PLATFORM expansion not yet supported", pathStr)
	}
	return pathStr, nil
}

// computeLibrarySearchOrder resolves any specified RPATH and RUNPATH values,
// before providing an ordered set of directories which should be searched for
// libraries satifying the link-time dependencies.
func (c *globalChecker) computeLibrarySearchOrder(elfPath string, depInfo info.ELFLinkDeps) ([]string, error) {
	var dirs []string
	for _, rpath := range depInfo.RPath {
		for _, p := range strings.Split(rpath, ":") {
			p, err := c.expandPathStr(elfPath, p)
			if err != nil {
				return nil, err
			}
			dirs = append(dirs, p)
		}
	}
	// If environment variables were set, LD_LIBRARY_PATH would be added here.
	for _, runpath := range depInfo.RunPath {
		for _, p := range strings.Split(runpath, ":") {
			p, err := c.expandPathStr(elfPath, p)
			if err != nil {
				return nil, err
			}
			dirs = append(dirs, p)
		}
	}
	for dir, _ := range c.libDirs {
		dirs = append(dirs, dir)
	}

	if !depInfo.Flags.NoDefaultLibs {
		// TODO: Are these the right paths?
		// TODO: This should be configurable.
		dirs = append(dirs, systemTrustedDirs...)
	}
	return dirs, nil
}

func (c *globalChecker) checkELFFile(elfPath string, r *vts.Resource, opts *vts.RunnerEnv) error {
	_, deps, _, interp, err := c.elfInfo(r, opts)
	if err != nil {
		return err
	}
	if err := c.checkInterp(interp, opts); err != nil {
		return err
	}

	searchDirs, err := c.computeLibrarySearchOrder(elfPath, deps)
	if err != nil {
		return err
	}

	for _, lib := range deps.Libs {
		if err := c.verifyDep(searchDirs, lib, deps, opts); err != nil {
			return fmt.Errorf("dep %s: %v", lib, err)
		}
	}

	// TODO: Check referenced symbols are present.
	return nil
}

func (c *globalChecker) libsInDir(dir string, opts *vts.RunnerEnv) (map[string]*vts.Resource, error) {
	if libs, ok := c.libsInDirCache[dir]; ok {
		return libs, nil
	}
	out := make(map[string]*vts.Resource, 8)
	for _, target := range opts.Universe.AllTargets() {
		if r, isResource := target.(*vts.Resource); isResource {
			parent := r.Parent.Target.(*vts.ResourceClass)
			switch parent.GlobalPath() {
			case "common://resources:support_files":
				path, err := resourcePath(r, opts)
				if err != nil {
					return nil, vts.WrapWithTarget(err, r)
				}
				if strings.HasPrefix(dir, path) {
					// Might be a library nested as a support file.
					files, err := opts.FS.ReadDir(dir)
					if err != nil {
						return nil, err
					}
					for _, f := range files {
						if strings.HasPrefix(f.Name(), "lib") || strings.Contains(f.Name(), ".so") {
							// Make fake library targets.
							pAttr, err := opts.Universe.Inject(&vts.Attr{
								Parent: vts.TargetRef{Path: "common://attrs:path"},
								Val:    starlark.String(filepath.Join(dir, f.Name())),
							})
							if err != nil {
								return nil, fmt.Errorf("injecting path attr for %s: %v", f.Name(), err)
							}
							fakeLib, err := opts.Universe.Inject(&vts.Resource{
								Parent:  vts.TargetRef{Path: "common://resources:sys_library"},
								Details: []vts.TargetRef{{Target: pAttr}},
							})
							if err != nil {
								return nil, fmt.Errorf("injecting %s: %v", f.Name(), err)
							}

							out[filepath.Join(dir, f.Name())] = fakeLib.(*vts.Resource)
						}
					}
				}
			case "common://resources:sys_library", "common://resources:sys_library_symlink":
				path, err := resourcePath(r, opts)
				if err != nil {
					return nil, vts.WrapWithTarget(err, r)
				}
				if strings.HasPrefix(path, dir) {
					out[path] = r
				}
			}
		}
	}

	c.libsInDirCache[dir] = out
	return out, nil
}

func (c *globalChecker) verifyDep(searchDirs []string, lib string, deps info.ELFLinkDeps, opts *vts.RunnerEnv) error {
	for _, dir := range searchDirs {
		libs, err := c.libsInDir(dir, opts)
		if err != nil {
			return err
		}
		for path, r := range libs {
			if filepath.Base(path) == lib {
				// TODO: Skip things like hwdep, arch, ABI compatibility.
				return c.checkDependentLib(path, r, opts)
			}
		}
	}
	return errors.New("required library was missing")
}

func (c *globalChecker) checkDependentLib(libPath string, r *vts.Resource, opts *vts.RunnerEnv) error {
	return c.checkELFFile(libPath, r, opts)
}
